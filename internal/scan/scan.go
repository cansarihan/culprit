// Package scan walks a repository's history, scores every commit, and rolls
// the verdicts up into the views the report and panel show: a ranked list, a
// weekly timeline, the files that keep getting quietly fixed, and who fixes them.
package scan

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/patch"
	"github.com/cansarihan/culprit/internal/score"
)

// Week is one bucket of the timeline.
type Week struct {
	Start    time.Time `json:"start"`
	Commits  int       `json:"commits"`
	Likely   int       `json:"likely"`
	Possible int       `json:"possible"`
	Silent   int       `json:"silent"`
}

// Hotspot is a file and how many flagged commits touched it.
type Hotspot struct {
	Path    string `json:"path"`
	Flagged int    `json:"flagged"`
	Silent  int    `json:"silent"`
}

// Author is a committer and their flagged commits.
type Author struct {
	Name    string `json:"name"`
	Commits int    `json:"commits"`
	Flagged int    `json:"flagged"`
}

// Report is a whole history scan.
type Report struct {
	Repo        string             `json:"repo"`
	Scanned     int                `json:"scanned"`
	Elapsed     time.Duration      `json:"elapsed"`
	Counts      map[string]int     `json:"counts"`
	Silent      int                `json:"silent"`
	Assessments []score.Assessment `json:"assessments"`
	Timeline    []Week             `json:"timeline"`
	Hotspots    []Hotspot          `json:"hotspots"`
	Authors     []Author           `json:"authors"`
}

// Run scans the repository at dir. progress, when not nil, is called after each
// commit is scored with the count done and the total.
func Run(ctx context.Context, dir string, opts gitlog.Options, progress func(done, total int)) (Report, error) {
	start := time.Now()
	commits, err := gitlog.Log(ctx, dir, opts)
	if err != nil {
		return Report{}, err
	}

	assessments := make([]score.Assessment, len(commits))
	errs := make([]error, len(commits))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				files, showErr := patch.Show(ctx, dir, commits[i].Hash)
				if showErr != nil {
					errs[i] = showErr
					continue
				}
				assessments[i] = score.Assess(commits[i], files)
				if progress != nil {
					mu.Lock()
					done++
					progress(done, len(commits))
					mu.Unlock()
				}
			}
		}()
	}
	for i := range commits {
		select {
		case jobs <- i:
		case <-ctx.Done():
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	for _, e := range errs {
		if e != nil {
			return Report{}, e
		}
	}

	report := Report{Repo: dir, Scanned: len(commits), Counts: map[string]int{}}
	for _, a := range assessments {
		report.Counts[a.Band]++
		if a.Silent {
			report.Silent++
		}
	}
	report.Timeline = timeline(assessments)
	report.Hotspots = hotspots(assessments)
	report.Authors = authors(assessments)

	sort.SliceStable(assessments, func(i, j int) bool { return assessments[i].Score > assessments[j].Score })
	report.Assessments = assessments
	report.Elapsed = time.Since(start).Round(time.Millisecond)
	return report, nil
}

func flagged(a score.Assessment) bool { return a.Band != score.BandUnlikely }

func weekStart(t time.Time) time.Time {
	t = t.UTC().Truncate(24 * time.Hour)
	offset := (int(t.Weekday()) + 6) % 7 // Monday-based weeks
	return t.AddDate(0, 0, -offset)
}

func timeline(as []score.Assessment) []Week {
	byWeek := map[time.Time]*Week{}
	for _, a := range as {
		ws := weekStart(a.Commit.When)
		w, ok := byWeek[ws]
		if !ok {
			w = &Week{Start: ws}
			byWeek[ws] = w
		}
		w.Commits++
		switch a.Band {
		case score.BandLikely:
			w.Likely++
		case score.BandPossible:
			w.Possible++
		}
		if a.Silent {
			w.Silent++
		}
	}
	out := make([]Week, 0, len(byWeek))
	for _, w := range byWeek {
		out = append(out, *w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

func hotspots(as []score.Assessment) []Hotspot {
	byPath := map[string]*Hotspot{}
	for _, a := range as {
		if !flagged(a) {
			continue
		}
		for _, p := range a.Paths {
			h, ok := byPath[p]
			if !ok {
				h = &Hotspot{Path: p}
				byPath[p] = h
			}
			h.Flagged++
			if a.Silent {
				h.Silent++
			}
		}
	}
	out := make([]Hotspot, 0, len(byPath))
	for _, h := range byPath {
		out = append(out, *h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Flagged != out[j].Flagged {
			return out[i].Flagged > out[j].Flagged
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func authors(as []score.Assessment) []Author {
	byName := map[string]*Author{}
	for _, a := range as {
		au, ok := byName[a.Commit.Author]
		if !ok {
			au = &Author{Name: a.Commit.Author}
			byName[a.Commit.Author] = au
		}
		au.Commits++
		if flagged(a) {
			au.Flagged++
		}
	}
	out := make([]Author, 0, len(byName))
	for _, au := range byName {
		out = append(out, *au)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Flagged != out[j].Flagged {
			return out[i].Flagged > out[j].Flagged
		}
		return out[i].Name < out[j].Name
	})
	return out
}
