package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/patch"
	"github.com/cansarihan/culprit/internal/score"
)

func runShow(args []string) error {
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return errors.New("show needs a repository and a commit: culprit show ./repo <commit>")
	}
	ctx, stop := signalContext()
	defer stop()
	repo, rev := flags.Arg(0), flags.Arg(1)

	commits, err := gitlog.Log(ctx, repo, gitlog.Options{Limit: 1, Rev: rev})
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		return fmt.Errorf("commit %s not found", rev)
	}
	files, err := patch.Show(ctx, repo, commits[0].Hash)
	if err != nil {
		return err
	}
	a := score.Assess(commits[0], files)

	kind := "announced"
	if a.Silent {
		kind = "silent"
	}
	fmt.Printf("%s  %s\n%s · %s\n\nscore %d · %s · %s fix · %d file(s), %d line(s)\n\n",
		a.Commit.Hash[:8], a.Commit.Subject, a.Commit.Author, a.Commit.When.Format("2006-01-02"),
		a.Score, a.Band, kind, a.Files, a.Churn)
	if len(a.Hits) == 0 {
		fmt.Println("no security signals")
		return nil
	}
	for _, h := range a.Hits {
		where := ""
		if h.File != "" && h.Signal.Family != "path" {
			where = "  (" + h.File + ")"
		}
		fmt.Printf("  +%-3d %-8s %-26s %s%s\n", h.Signal.Weight, h.Signal.Family, h.Signal.Label, strings.TrimSpace(h.Evidence), where)
	}
	return nil
}
