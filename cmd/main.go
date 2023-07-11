package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/clarete/langlang/go"
)

const defaultWritePermission = 0644 // -rw-r--r--

func main() {
	var (
		grammarPath = flag.String("grammar", "", "Path to the grammar file")
		language    = flag.String("language", "", "Output language")
		outputPath  = flag.String("output", "", "Output path")
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
	var (
		extension  string
		outputData string
	)

	log.Printf("AST: %s\n", ast)

	switch *language {
	// case "go":
	// 	extension = ".go"
	// 	outputData, err = parsing.GenParserGo(ast)

	// case "python":
	// 	extension = ".py"
	// 	outputData, err = parsing.GenParserPython(ast)
	default:
		log.Fatalf("Output language `%s` not supported", *language)
	}
	if err != nil {
		log.Fatalf("Can't emit code: %s", err.Error())
	}

	base, ext := filepath.Base(*grammarPath), filepath.Ext(*grammarPath)
	fileName := strings.TrimSuffix(base, ext)
	outputFile := filepath.Join(*outputPath, fileName+extension)

	err = os.WriteFile(outputFile, []byte(outputData), defaultWritePermission)
	if err != nil {
		log.Fatalf("Can't write output file: %s", err.Error())
	}
}
