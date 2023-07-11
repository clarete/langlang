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
	log.Printf("AST: %s\n", ast)

	var outputData string
	switch *language {
	case "go":
		outputData, err = parsing.GenGo(ast)

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
