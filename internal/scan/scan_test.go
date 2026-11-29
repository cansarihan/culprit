package scan

import (
	"context"
	"os/exec"
	"testing"

	"github.com/cansarihan/culprit/internal/demo"
	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/score"
)

func demoReport(t *testing.T) Report {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	if err := demo.Build(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	r, err := Run(context.Background(), dir, gitlog.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func find(r Report, subject string) score.Assessment {
	for _, a := range r.Assessments {
		if a.Commit.Subject == subject {
			return a
		}
	}
	return score.Assessment{}
}

func TestDemoHistory(t *testing.T) {
	r := demoReport(t)
	if r.Scanned != demo.Commits() {
		t.Fatalf("scanned %d, want %d", r.Scanned, demo.Commits())
	}
	for _, subject := range []string{"tidy frame reader", "refactor session check", "small fix", "limit upload size"} {
		a := find(r, subject)
		if a.Band != score.BandLikely || !a.Silent {
			t.Errorf("%q: band=%s silent=%v score=%d, want a likely silent fix", subject, a.Band, a.Silent, a.Score)
		}
	}
	cve := find(r, "fix SQL injection in user lookup (CVE-2025-31337)")
	if cve.Band != score.BandLikely || cve.Silent {
		t.Errorf("CVE commit: band=%s silent=%v, want likely and announced", cve.Band, cve.Silent)
	}
	for _, subject := range []string{"accept uploads", "docs: changelog", "initial commit"} {
		if a := find(r, subject); a.Band != score.BandUnlikely {
			t.Errorf("%q flagged as %s (score %d)", subject, a.Band, a.Score)
		}
	}
	if r.Silent != 4 {
		t.Errorf("silent = %d, want 4", r.Silent)
	}
	if r.Assessments[0].Score < r.Assessments[len(r.Assessments)-1].Score {
		t.Error("assessments are not ranked")
	}
}

func TestRollups(t *testing.T) {
	r := demoReport(t)
	if len(r.Timeline) < 10 {
		t.Errorf("timeline has %d weeks", len(r.Timeline))
	}
	total := 0
	for i, w := range r.Timeline {
		total += w.Commits
		if i > 0 && !r.Timeline[i-1].Start.Before(w.Start) {
			t.Fatal("timeline is not in order")
		}
	}
	if total != r.Scanned {
		t.Errorf("timeline counts %d commits, want %d", total, r.Scanned)
	}
	if len(r.Hotspots) == 0 || r.Hotspots[0].Flagged == 0 {
		t.Errorf("hotspots = %+v", r.Hotspots)
	}
	if len(r.Authors) != 3 {
		t.Errorf("authors = %d, want 3", len(r.Authors))
	}
}

func TestProgressAndCancel(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	if err := demo.Build(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	last := 0
	if _, err := Run(context.Background(), dir, gitlog.Options{Limit: 5}, func(done, total int) { last = done }); err != nil {
		t.Fatal(err)
	}
	if last != 5 {
		t.Errorf("progress reached %d, want 5", last)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, dir, gitlog.Options{}, nil); err == nil {
		t.Error("a cancelled scan should fail")
	}
}
