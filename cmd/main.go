package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/clarete/langlang/go"
)

const defaultWritePermission = 0644 // -rw-r--r--

func main() {
	var (
		grammarPath = flag.String("grammar", "", "Path to the grammar file")
		language    = flag.String("language", "", "Output language")
		astOnly     = flag.Bool("ast-only", false, "Output the AST of the grammar")

		// options specific to the go generator
		goOptPackage      = flag.String("go-package", "parser", "Name of the go package in the generated parser")
		goOptStructSuffix = flag.String("go-struct-suffix", "", "Suffix of the Go struct generated for the parser")
	)
	flag.Parse()

	if *language == "" {
		*language = "go"
	}

	if *grammarPath == "" {
		log.Fatal("Grammar not informed")
	}

	grammarData, err := os.ReadFile(*grammarPath)
	if err != nil {
		log.Fatalf("Can't read grammar file: %s", err.Error())
	}

	parser := parsing.NewGrammarParser(string(grammarData))
	ast, err := parser.Parse()
	if err != nil {
		log.Fatalf("Can't parse grammar file: %s", err.Error())
	}
	if *astOnly {
		log.Println(ast)
		return
	}

	var outputData string
	switch *language {
	case "go":
		outputData, err = parsing.GenGo(ast, parsing.GenGoOptions{
			PackageName:  *goOptPackage,
			StructSuffix: *goOptStructSuffix,
		})

	// case "python":
	// 	outputData, err = parsing.GenParserPython(ast)
	default:
		log.Fatalf("Output language `%s` not supported", *language)
	}
	if err != nil {
		log.Fatalf("Can't emit code: %s", err.Error())
	}

	fmt.Println(outputData)
}
