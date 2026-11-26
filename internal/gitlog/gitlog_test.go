package gitlog

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	data := []byte("abc\x1fAda\x1fada@x\x1f1700000000\x1ffix parser\x1fbody line\x1e\n" +
		"def\x1fBob\x1fbob@x\x1f1700000100\x1fadd docs\x1f\x1e\n")
	commits, err := parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 || commits[0].Subject != "fix parser" || commits[0].Body != "body line" {
		t.Fatalf("parsed %+v", commits)
	}
	if commits[1].When.Unix() != 1700000100 {
		t.Errorf("timestamp = %v", commits[1].When)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := parse([]byte("only\x1ftwo\x1e")); err == nil {
		t.Fatal("expected an error for a short record")
	}
}

// initRepo builds a throwaway repository with two commits.
func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	for i, msg := range []string{"first", "second"} {
		if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte{byte('a' + i)}, 0o600); err != nil {
			t.Fatal(err)
		}
		git("add", ".")
		git("commit", "-q", "-m", msg)
	}
	return dir
}

func TestLogRealRepo(t *testing.T) {
	dir := initRepo(t)
	commits, err := Log(context.Background(), dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 || commits[0].Subject != "second" {
		t.Fatalf("log = %+v", commits)
	}
	limited, _ := Log(context.Background(), dir, Options{Limit: 1})
	if len(limited) != 1 {
		t.Errorf("limit ignored: %d", len(limited))
	}
}
