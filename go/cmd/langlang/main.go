package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/clarete/langlang/go"
)

const defaultWritePermission = 0644 // -rw-r--r--

type args struct {
	grammarPath *string

	grammarAST *bool
	grammarASM *bool

	outputLang *string
	outputPath *string

	disableBuiltins           *bool
	disableWhitespaceHandling *bool
	disableCaptures           *bool
	enableCaptureSpacing      *bool

	inputPath   *string
	interactive *bool

	goOptPackage   *string
	goOptRemoveLib *bool
}

func readArgs() *args {
	a := &args{
		grammarPath: flag.String("grammar", "", "Path to the grammar file"),

		// Debugging Options

		grammarAST: flag.Bool("grammar-ast", false, "Output the AST of the grammar"),
		grammarASM: flag.Bool("grammar-asm", false, "Output the ASM of the grammar"),

		// Output Options

		disableWhitespaceHandling: flag.Bool("disable-whitespace-handling", false, "Tells the compiler not to inject automatic white space char handling into the grammar"),
		disableBuiltins:           flag.Bool("disable-builtins", false, "Tells the compiler not to inject builtin rules into the grammar"),
		disableCaptures:           flag.Bool("disable-captures", false, "Tells the compiler not to inject capture rules into the grammar"),
		enableCaptureSpacing:      flag.Bool("enable-capture-spacing", false, "If enabled, the runtime will capture the output of the Spacing production"),

		// Dynamic parser generation and evaluation

		inputPath: flag.String("input", "", "Path to the input file"),

		// AOT parser generation

		outputPath: flag.String("output-path", "", "Path to the output file"),
		outputLang: flag.String("output-language", "", "Output language"),

		// options specific to the go generator

		goOptPackage:   flag.String("go-package", "parser", "Name of the go package in the generated parser"),
		goOptRemoveLib: flag.Bool("go-remove-lib", false, "Include lib in the output parser"),
	}

	flag.Parse()

	return a
}

func main() {
	var (
		a         = readArgs()
		suppress  []string
		errLabels map[string]string
	)

	if *a.grammarPath == "" {
		log.Fatal("Grammar not informed")
	}

	// TODO: this should move into the API

	ast, err := importGrammar(*a.grammarPath)
	if err != nil {
		log.Fatal(err)
	}

	// Post process the AST
	// TODO: this should move into the API

	if !*a.disableBuiltins {
		ast, err = langlang.AddBuiltins(ast)
		if err != nil {
			log.Fatal(err)
		}
	}

	if !*a.disableWhitespaceHandling {
		ast, err = langlang.InjectWhitespaces(ast)
		if err != nil {
			log.Fatal(err)
		}
	}

	if !*a.disableCaptures {
		ast, err = langlang.AddCaptures(ast)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *a.grammarAST {
		fmt.Println(ast.HighlightPrettyString())
		return
	}

	if !*a.enableCaptureSpacing {
		suppress = []string{"Spacing"}
	}

	// Translate the AST into bytecode

	asm, err := langlang.Compile(ast, langlang.CompilerConfig{Optimize: 1})
	if err != nil {
		log.Fatal(err)
	}

	if *a.grammarASM {
		fmt.Println(asm.HighlightPrettyString())
		return
	}

	// If it's interactive, it will open a lil REPL shell

	if *a.inputPath == "" {
		code := langlang.Encode(asm)

		for {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("> ")
			text, _ := reader.ReadString('\n')

			if text == "" {
				fmt.Println("")
				break
			}

			if text == "\n" {
				continue
			}

			val, _, err := code.MatchE(strings.NewReader(text), errLabels, suppress)
			if err != nil {
				fmt.Println("ERROR: " + err.Error())
			} else if val != nil {
				fmt.Println(val.HighlightPrettyString())
			}
		}
		return
	}

	// if there's an input path, just run the match right away

	if *a.inputPath != "" {
		text, err := os.ReadFile(*a.inputPath)
		if err != nil {
			log.Fatalf("Can't open input file: %s", err.Error())
		}
		code := langlang.Encode(asm)
		val, _, err := code.MatchE(strings.NewReader(string(text)), errLabels, suppress)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		} else if val != nil {
			fmt.Println(val.HighlightPrettyString())
		}
		return
	}

	// Configure output options

	if *a.outputLang == "" {
		log.Fatalf("Expected `-output-language`")
	}

	var outputData string
	switch *a.outputLang {
	case "go":
		outputData, err = langlang.GenGo(ast, langlang.GenGoOptions{
			PackageName: *a.goOptPackage,
			RemoveLib:   *a.goOptRemoveLib,
		})
	case "goeval":
		outputData, err = langlang.GenGoEval(asm, langlang.GenGoOptions{
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
