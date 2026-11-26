// Package patch reads the diff a commit introduced and splits it per file into
// the lines it added and removed — the raw material for the diff signals.
package patch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// FileDiff is what one commit changed in one file.
type FileDiff struct {
	Path    string   `json:"path"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}

// Churn is the number of lines touched.
func (f FileDiff) Churn() int { return len(f.Added) + len(f.Removed) }

// Show returns the per-file diff of a commit, with no context lines so every
// line seen is a real change.
func Show(ctx context.Context, dir, hash string) ([]FileDiff, error) {
	cmd := exec.CommandContext(ctx, "git", "show", "--format=", "--unified=0", "--no-color", "--no-ext-diff", hash)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git show %s: %w: %s", hash, err, strings.TrimSpace(stderr.String()))
	}
	return Parse(stdout.Bytes()), nil
}

// Parse splits unified diff output into per-file changes. Binary files carry no
// line changes and are reported with empty added and removed sets.
func Parse(data []byte) []FileDiff {
	var files []FileDiff
	var current *FileDiff
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			files = append(files, FileDiff{Path: pathFromHeader(line)})
			current = &files[len(files)-1]
		case current == nil:
			continue
		case strings.HasPrefix(line, "+++ "):
			if p := strings.TrimPrefix(line, "+++ "); p != "/dev/null" {
				current.Path = strings.TrimPrefix(p, "b/")
			}
		case strings.HasPrefix(line, "--- "):
			continue
		case strings.HasPrefix(line, "+"):
			current.Added = append(current.Added, line[1:])
		case strings.HasPrefix(line, "-"):
			current.Removed = append(current.Removed, line[1:])
		}
	}
	return files
}

// pathFromHeader takes the b/ side of a "diff --git a/x b/x" line.
func pathFromHeader(line string) string {
	if i := strings.LastIndex(line, " b/"); i >= 0 {
		return line[i+3:]
	}
	return strings.TrimPrefix(line, "diff --git ")
}
