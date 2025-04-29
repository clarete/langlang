package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/clarete/langlang/go"
)

const defaultWritePermission = 0644 // -rw-r--r--

type args struct {
	grammarPath *string

	astOnly *bool

	outputLang *string
	outputPath *string

	disableWhitespaceHandling *bool

	goOptPackage   *string
	goOptRemoveLib *bool
}

func readArgs() *args {
	a := &args{
		grammarPath: flag.String("grammar", "", "Path to the grammar file"),

		// Debugging Options

		astOnly: flag.Bool("ast-only", false, "Output the AST of the grammar"),

		// Output Options

		disableWhitespaceHandling: flag.Bool("disable-whitespace-handling", false, "Inject whitespace handling rules into the grammar"),

		// AOT parser generation

		outputPath: flag.String("output-path", "/dev/stdout", "Path to the output file"),
		outputLang: flag.String("output-language", "", "Output language"),

		// options specific to the go generator

		goOptPackage:   flag.String("go-package", "parser", "Name of the go package in the generated parser"),
		goOptRemoveLib: flag.Bool("go-remove-lib", false, "Include lib in the output parser"),
	}

	flag.Parse()

	return a
}

func main() {
	a := readArgs()

	if *a.grammarPath == "" {
		log.Fatal("Grammar not informed")
	}

	ast, err := importGrammar(*a.grammarPath)
	if err != nil {
		log.Fatal(err)
	}

	// Post process the AST

	if !*a.disableWhitespaceHandling {
		ast, err = langlang.InjectWhitespaces(ast)
		if err != nil {
			log.Fatal(err)
		}
	}

	ast, err = langlang.AddBuiltins(ast)
	if err != nil {
		log.Fatal(err)
	}

	if *a.astOnly {
		fmt.Println(ast.PrettyPrint())
		return
	}

	// Configure output options

	if *a.outputLang == "" {
		*a.outputLang = "go"
	}

	var outputData string
	switch *a.outputLang {
	case "go":
		outputData, err = langlang.GenGo(ast, langlang.GenGoOptions{
			PackageName: *a.goOptPackage,
			RemoveLib:   *a.goOptRemoveLib,
		})

	// case "python":
	// 	outputData, err = langlang.GenParserPython(ast)
	default:
		log.Fatalf("Output language `%s` not supported", *a.outputLang)
	}
	if err != nil {
		log.Fatalf("Can't emit code: %s", err.Error())
	}

	if err = os.WriteFile(*a.outputPath, []byte(outputData), 0644); err != nil {
		log.Fatalf("Can't write output: %s", err.Error())
	}
}

func importGrammar(path string) (langlang.AstNode, error) {
	importLoader := langlang.NewRelativeImportLoader()
	importResolver := langlang.NewImportResolver(importLoader)
	return importResolver.Resolve(path)
}
