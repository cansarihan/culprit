package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/cansarihan/culprit/internal/demo"
)

func runDemo(args []string) error {
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("demo needs a target directory: culprit demo ./relay")
	}
	ctx, stop := signalContext()
	defer stop()
	if err := demo.Build(ctx, flags.Arg(0)); err != nil {
		return err
	}
	fmt.Printf("wrote a %d-commit history to %s\n", demo.Commits(), flags.Arg(0))
	return nil
}
