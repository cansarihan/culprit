// Command culprit reads a repository's history and points at the commits that
// look like security fixes — above all the ones that never said so.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/report"
	"github.com/cansarihan/culprit/internal/scan"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "culprit: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usage()
	}
	args := os.Args[2:]
	switch os.Args[1] {
	case "scan":
		return runScan(args)
	case "show":
		return runShow(args)
	case "signals":
		return runSignals(args)
	case "demo":
		return runDemo(args)
	case "serve":
		return runServe(args)
	case "version":
		fmt.Printf("culprit %s (commit %s, built %s)\n", version, commit, date)
		return nil
	default:
		return usage()
	}
}

func usage() error {
	fmt.Fprintln(os.Stderr, "usage: culprit <scan|show|signals|demo|serve|version> [flags] [repo]")
	fmt.Fprintln(os.Stderr, "  culprit scan -since '1 year ago' ./repo")
	fmt.Fprintln(os.Stderr, "  culprit show ./repo <commit>")
	fmt.Fprintln(os.Stderr, "  culprit serve -repo ./repo")
	return errors.New("no command given")
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

func repoArg(flags *flag.FlagSet) string {
	if flags.NArg() > 0 {
		return flags.Arg(0)
	}
	return "."
}

func runScan(args []string) error {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	since := flags.String("since", "", "only commits newer than this (anything git log --since accepts)")
	limit := flags.Int("n", 0, "only the most recent N commits")
	rev := flags.String("rev", "", "revision to walk from (default HEAD)")
	all := flags.Bool("all", false, "list unflagged commits too")
	asJSON := flags.Bool("json", false, "print JSON")
	asMarkdown := flags.Bool("md", false, "print a Markdown digest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	ctx, stop := signalContext()
	defer stop()

	r, err := scan.Run(ctx, repoArg(flags), gitlog.Options{Since: *since, Limit: *limit, Rev: *rev}, nil)
	if err != nil {
		return err
	}
	switch {
	case *asJSON:
		return report.JSON(os.Stdout, r)
	case *asMarkdown:
		report.Markdown(os.Stdout, r)
	default:
		report.Table(os.Stdout, r, *all)
	}
	return nil
}
