// Package report renders a history scan for the terminal, as JSON, or as a
// Markdown digest that can be pasted into an issue or a review.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/cansarihan/culprit/internal/scan"
	"github.com/cansarihan/culprit/internal/score"
)

// JSON writes any value as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Table writes the flagged commits, worst first. With all set, unflagged
// commits are listed too.
func Table(w io.Writer, r scan.Report, all bool) {
	fmt.Fprintf(w, "%s\n%d commits · %d likely · %d possible · %d silent fixes\n\n",
		r.Repo, r.Scanned, r.Counts[score.BandLikely], r.Counts[score.BandPossible], r.Silent)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SCORE\tBAND\tKIND\tCOMMIT\tDATE\tSUBJECT\tEVIDENCE")
	rows := 0
	for _, a := range r.Assessments {
		if !all && a.Band == score.BandUnlikely {
			continue
		}
		rows++
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", a.Score, a.Band, kind(a), short(a.Commit.Hash),
			a.Commit.When.Format("2006-01-02"), clip(a.Commit.Subject, 48), labels(a))
	}
	_ = tw.Flush()
	if rows == 0 {
		fmt.Fprintln(w, "no commits look like security fixes")
	}
}

// Markdown writes a digest of the flagged commits.
func Markdown(w io.Writer, r scan.Report) {
	fmt.Fprintf(w, "# Culprit report\n\n`%s` — %d commits scanned, **%d silent fixes**.\n\n", r.Repo, r.Scanned, r.Silent)
	fmt.Fprintln(w, "| Score | Band | Kind | Commit | Subject | Evidence |")
	fmt.Fprintln(w, "|---:|---|---|---|---|---|")
	for _, a := range r.Assessments {
		if a.Band == score.BandUnlikely {
			continue
		}
		fmt.Fprintf(w, "| %d | %s | %s | `%s` | %s | %s |\n", a.Score, a.Band, kind(a), short(a.Commit.Hash),
			strings.ReplaceAll(a.Commit.Subject, "|", `\|`), labels(a))
	}
	if len(r.Hotspots) > 0 {
		fmt.Fprintln(w, "\n## Hotspots\n\n| File | Flagged | Silent |\n|---|---:|---:|")
		for _, h := range r.Hotspots {
			fmt.Fprintf(w, "| `%s` | %d | %d |\n", h.Path, h.Flagged, h.Silent)
		}
	}
}

func kind(a score.Assessment) string {
	switch {
	case a.Band == score.BandUnlikely:
		return "-"
	case a.Silent:
		return "silent"
	default:
		return "announced"
	}
}

func labels(a score.Assessment) string {
	out := make([]string, 0, len(a.Hits))
	for _, h := range a.Hits {
		out = append(out, h.Signal.Label)
	}
	return strings.Join(out, ", ")
}

func short(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
