package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cansarihan/culprit/internal/report"
	"github.com/cansarihan/culprit/internal/signals"
)

func runSignals(args []string) error {
	flags := flag.NewFlagSet("signals", flag.ContinueOnError)
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *asJSON {
		return report.JSON(os.Stdout, signals.Catalog())
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FAMILY\tWEIGHT\tSIGNAL\tMEANING")
	for _, s := range signals.Catalog() {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", s.Family, s.Weight, s.Label, s.Description)
	}
	return tw.Flush()
}
