package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/clarete/langlang/go/extract"
)

func runExtract(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	grammar := fs.String("grammar", "", "Path to the grammar file")
	source := fs.String("source", "", "Path to Go source file (defaults to $GOFILE)")
	fs.Parse(args)

	if *grammar == "" {
		fmt.Fprintf(os.Stderr, "error: -grammar is required\n")
		os.Exit(1)
	}

	goFile := *source
	if goFile == "" {
		goFile = os.Getenv("GOFILE")
	}
	if goFile == "" {
		fmt.Fprintf(os.Stderr, "error: -source or $GOFILE is required\n")
		os.Exit(1)
	}

	if err := extract.Generate(goFile, *grammar); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
