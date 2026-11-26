// Package gitlog reads a repository's history through the git command line,
// returning each commit's identity and message in a form the scorer can read.
package gitlog

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Commit is one entry of the history.
type Commit struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Email   string    `json:"email"`
	When    time.Time `json:"when"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
}

// Options narrows the history that is read.
type Options struct {
	Limit int    // most recent N commits; zero reads everything
	Since string // anything git accepts for --since, e.g. "6 months ago"
	Rev   string // revision to walk from; empty means HEAD
}

const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
	format    = "%H" + fieldSep + "%an" + fieldSep + "%ae" + fieldSep + "%at" + fieldSep + "%s" + fieldSep + "%b" + recordSep
)

// Log returns the commits of the repository at dir, newest first. Merge commits
// are skipped: their diff repeats work already scored on the branch.
func Log(ctx context.Context, dir string, opts Options) ([]Commit, error) {
	args := []string{"log", "--no-merges", "--format=" + format}
	if opts.Limit > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Limit))
	}
	if opts.Since != "" {
		args = append(args, "--since="+opts.Since)
	}
	if opts.Rev != "" {
		args = append(args, opts.Rev, "--")
	}
	out, err := run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	return parse(out)
}

func parse(data []byte) ([]Commit, error) {
	var commits []Commit
	for _, record := range strings.Split(string(data), recordSep) {
		record = strings.TrimLeft(record, "\n")
		if strings.TrimSpace(record) == "" {
			continue
		}
		fields := strings.SplitN(record, fieldSep, 6)
		if len(fields) != 6 {
			return nil, fmt.Errorf("malformed log record: %q", record)
		}
		unix, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad timestamp %q: %w", fields[3], err)
		}
		commits = append(commits, Commit{
			Hash:    fields[0],
			Author:  fields[1],
			Email:   fields[2],
			When:    time.Unix(unix, 0).UTC(),
			Subject: fields[4],
			Body:    strings.TrimSpace(fields[5]),
		})
	}
	return commits, nil
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
