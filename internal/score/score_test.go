package score

import (
	"testing"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/patch"
)

func TestSilentFixScoresHigh(t *testing.T) {
	a := Assess(gitlog.Commit{Subject: "tidy frame reader"}, []patch.FileDiff{{
		Path:  "internal/decode/frame.go",
		Added: []string{"\tif n > len(buf) {", "\t\treturn errShort", "\t}", "\tr = http.MaxBytesReader(w, r, 1<<20)"},
	}})
	if a.Band != BandLikely {
		t.Fatalf("band = %s (score %d), want likely", a.Band, a.Score)
	}
	if !a.Silent {
		t.Error("no announcement: the fix should be silent")
	}
	if !a.Targeted {
		t.Error("a 4-line change is targeted")
	}
}

func TestAnnouncedFixIsNotSilent(t *testing.T) {
	a := Assess(gitlog.Commit{Subject: "fix CVE-2025-1234 in session handling"}, []patch.FileDiff{{
		Path: "auth/session.go", Added: []string{"if subtle.ConstantTimeCompare(a, b) != 1 {"},
	}})
	if a.Silent {
		t.Error("a CVE commit is announced, not silent")
	}
	if a.Band != BandLikely {
		t.Errorf("band = %s", a.Band)
	}
}

func TestPathAloneIsNotEvidence(t *testing.T) {
	a := Assess(gitlog.Commit{Subject: "rename variable"}, []patch.FileDiff{{
		Path: "auth/login.go", Added: []string{"x := y"}, Removed: []string{"z := y"},
	}})
	if a.Score != 0 || a.Band != BandUnlikely || a.Silent {
		t.Errorf("path-only commit scored %d (%s, silent=%v)", a.Score, a.Band, a.Silent)
	}
}

func TestScoreCapped(t *testing.T) {
	a := Assess(gitlog.Commit{Subject: "security: fix overflow, sanitize input, crash, CVE-2024-0001"}, []patch.FileDiff{{
		Path:  "auth/crypto/parse.go",
		Added: []string{"if n > len(b) {", "subtle.ConstantTimeCompare(a,b)", "html.EscapeString(s)", "filepath.Clean(p)"},
	}})
	if a.Score != 100 {
		t.Errorf("score = %d, want capped at 100", a.Score)
	}
}
