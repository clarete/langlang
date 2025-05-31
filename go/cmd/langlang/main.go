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

	astOnly     *bool
	asmOnly     *bool
	asmOptimize *int

	outputType *string
	outputLang *string
	outputPath *string

	disableBuiltins           *bool
	disableWhitespaceHandling *bool
	disableCaptureAll         *bool

	inputPath   *string
	interactive *bool

	goOptPackage   *string
	goOptRemoveLib *bool
}

func readArgs() *args {
	a := &args{
		grammarPath: flag.String("grammar", "", "Path to the grammar file"),

		// Debugging Options

		astOnly:     flag.Bool("ast-only", false, "Output the AST of the grammar"),
		asmOnly:     flag.Bool("asm-only", false, "Output the ASM of the grammar"),
		interactive: flag.Bool("interactive", false, "Drops into a shell"),

		// Output Options

		disableWhitespaceHandling: flag.Bool("disable-whitespace-handling", false, "Inject whitespace handling rules into the grammar"),
		disableBuiltins:           flag.Bool("disable-builtins", false, "Inject builtin rules into the grammar"),
		disableCaptureAll:         flag.Bool("disable-capture-all", false, "Inject capture rules into the grammar around every definition"),

		asmOptimize: flag.Int("asm-optimize", 0, "How much to optimize the ASM output [0-1]"),

		// Dynamic parser generation and evaluation

		inputPath: flag.String("input", "", "Path to the input file"),

		// AOT parser generation

		outputType: flag.String("output-type", "code", "What type of parser should we generate. Options: 'code' and 'bytecode'"),
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

	if !*a.disableBuiltins {
		ast, err = langlang.AddBuiltins(ast)
		if err != nil {
			log.Fatal(err)
		}
	}

	if !*a.disableCaptureAll {
		ast, err = langlang.AddCaptures(ast)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *a.astOnly {
		fmt.Println(ast.HighlightPrettyString())
		return
	}

	// Translate the AST into bytecode

	asm, err := langlang.Compile(ast, langlang.CompilerConfig{
		Optimize: *a.asmOptimize,
	})
	if err != nil {
		log.Fatal(err)
	}

	if *a.asmOnly {
		fmt.Println(asm.HighlightPrettyString())
		return
	}

	// If it's interactive, it will open a lil REPL shell

	if *a.interactive {
		fmt.Println(ast.HighlightPrettyString())
		fmt.Println(asm.HighlightPrettyString())
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

			val, _, err := code.Match(strings.NewReader(text))
			if err != nil {
				fmt.Println("ERROR: " + err.Error())
			} else {
				fmt.Printf("\n%v\n", val)
			}
			// fmt.Print(text)
		}
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
