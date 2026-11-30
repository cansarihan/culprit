package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/scan"
	"github.com/cansarihan/culprit/internal/score"
	"github.com/cansarihan/culprit/internal/signals"
)

func sample() scan.Report {
	hit := signals.Hit{Signal: signals.Signal{ID: "diff-bounds", Label: "new bounds check"}}
	return scan.Report{
		Repo: "./relay", Scanned: 3, Silent: 1,
		Counts: map[string]int{score.BandLikely: 1, score.BandUnlikely: 2},
		Assessments: []score.Assessment{
			{Commit: gitlog.Commit{Hash: "a1b2c3d4e5f6", Subject: "tidy frame | reader", When: time.Date(2025, 7, 16, 0, 0, 0, 0, time.UTC)},
				Score: 56, Band: score.BandLikely, Silent: true, Hits: []signals.Hit{hit}},
			{Commit: gitlog.Commit{Hash: "ffff0000aaaa", Subject: "docs"}, Band: score.BandUnlikely},
		},
		Hotspots: []scan.Hotspot{{Path: "decode/frame.go", Flagged: 1, Silent: 1}},
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, sample(), false)
	out := buf.String()
	for _, want := range []string{"1 silent fixes", "a1b2c3d4", "silent", "new bounds check", "2025-07-16"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "ffff0000") {
		t.Error("unflagged commit listed without -all")
	}
	buf.Reset()
	Table(&buf, sample(), true)
	if !strings.Contains(buf.String(), "ffff0000") {
		t.Error("-all should list every commit")
	}
}

func TestMarkdown(t *testing.T) {
	var buf bytes.Buffer
	Markdown(&buf, sample())
	out := buf.String()
	for _, want := range []string{"# Culprit report", "**1 silent fixes**", "`a1b2c3d4`", `tidy frame \| reader`, "## Hotspots", "`decode/frame.go`"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown missing %q\n%s", want, out)
		}
	}
}
