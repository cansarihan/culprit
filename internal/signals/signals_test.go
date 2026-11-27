package signals

import (
	"testing"

	"github.com/cansarihan/culprit/internal/patch"
)

func ids(hits []Hit) map[string]bool {
	out := map[string]bool{}
	for _, h := range hits {
		out[h.Signal.ID] = true
	}
	return out
}

func TestMessageHits(t *testing.T) {
	got := ids(MessageHits("reject malformed headers", "avoids an out-of-bounds read"))
	if !got["msg-input"] || !got["msg-memory"] {
		t.Errorf("missed message signals: %v", got)
	}
	if got["msg-cve"] {
		t.Error("no CVE in this message")
	}
	if !ids(MessageHits("Fix CVE-2024-12345", ""))["msg-cve"] {
		t.Error("CVE id not detected")
	}
	if !ids(MessageHits("fix", ""))["msg-vague"] {
		t.Error("a bare 'fix' should be vague")
	}
	if ids(MessageHits("fix the parser so it stops leaking", ""))["msg-vague"] {
		t.Error("a descriptive message is not vague")
	}
}

func TestDiffHits(t *testing.T) {
	files := []patch.FileDiff{{
		Path:    "internal/decode/frame.go",
		Added:   []string{"\tif n > len(buf) {", "\t\treturn errShort", "\tif subtle.ConstantTimeCompare(got, want) != 1 {"},
		Removed: []string{"\tif got == want {"},
	}}
	got := ids(DiffHits(files))
	if !got["diff-bounds"] || !got["diff-consttime"] {
		t.Errorf("missed diff signals: %v", got)
	}
}

func TestDiffSwaps(t *testing.T) {
	files := []patch.FileDiff{{
		Path:    "store/users.go",
		Removed: []string{`	rows, err := db.Query(fmt.Sprintf("SELECT * FROM users WHERE id=%s", id))`, `import "math/rand"`},
		Added:   []string{`	rows, err := db.Query("SELECT * FROM users WHERE id=$1", id)`, `import "crypto/rand"`},
	}}
	got := ids(DiffHits(files))
	if !got["diff-sqlparam"] || !got["diff-cryptorand"] {
		t.Errorf("missed swap signals: %v", got)
	}
}

func TestDiffIgnoresTests(t *testing.T) {
	files := []patch.FileDiff{{Path: "frame_test.go", Added: []string{"if n > len(buf) {"}}}
	if len(DiffHits(files)) != 0 {
		t.Error("test files should not trip diff signals")
	}
}

func TestPathHits(t *testing.T) {
	got := ids(PathHits([]patch.FileDiff{{Path: "pkg/auth/session.go"}, {Path: "README.md"}}))
	if !got["path-auth"] {
		t.Errorf("missed path signal: %v", got)
	}
}

func TestCatalogComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Catalog() {
		if seen[s.ID] {
			t.Errorf("duplicate signal %q", s.ID)
		}
		seen[s.ID] = true
		if s.Weight <= 0 || s.Description == "" {
			t.Errorf("signal %q needs a weight and description", s.ID)
		}
	}
	if len(seen) < 15 {
		t.Errorf("catalog has %d signals", len(seen))
	}
}
