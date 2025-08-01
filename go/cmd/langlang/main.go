package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/clarete/langlang/go"
)

type args struct {
	grammarPath *string

	grammarAST *bool
	grammarASM *bool

	outputLang *string
	outputPath *string

	disableBuiltins      *bool
	disableSpaces        *bool
	disableCharsets      *bool
	disableCaptures      *bool
	disableCaptureSpaces *bool
	suppressSpaces       *bool

	showFails *bool

	inputPath   *string
	interactive *bool

	goOptPackage   *string
	goOptParser    *string
	goOptRemoveLib *bool
}

func readArgs() *args {
	a := &args{
		grammarPath: flag.String("grammar", "", "Path to the grammar file"),

		// Debugging Options

		grammarAST: flag.Bool("grammar-ast", false, "Output the AST of the grammar"),
		grammarASM: flag.Bool("grammar-asm", false, "Output the ASM of the grammar"),

		// Output Options

		disableBuiltins:      flag.Bool("disable-builtins", false, "Tells the compiler not to inject builtin rules into the grammar"),
		disableSpaces:        flag.Bool("disable-spaces", false, "Tells the compiler not to inject automatic whitespace char handling into the grammar"),
		disableCharsets:      flag.Bool("disable-charsets", false, "Inject whitespace handling rules into the grammar"),
		disableCaptures:      flag.Bool("disable-captures", false, "Tells the compiler not to inject capture rules into the grammar"),
		disableCaptureSpaces: flag.Bool("disable-capture-spaces", false, "Tells the compiler not to inject capture rules for spaces into the grammar"),
		suppressSpaces:       flag.Bool("suppress-spaces", true, "If enabled, it will suppress capturing spaces during Runtime"),
		showFails:            flag.Bool("show-fails", true, "If enabled, shows what the parser attempted to match (there is a perf penalty cost for this)"),

		// Dynamic parser generation and evaluation

		inputPath: flag.String("input", "", "Path to the input file"),

		// AOT parser generation

		outputPath: flag.String("output-path", "", "Path to the output file"),
		outputLang: flag.String("output-language", "", "Output language"),

		// options specific to the go generator

		goOptPackage:   flag.String("go-package", "parser", "Name of the go package in the generated parser"),
		goOptParser:    flag.String("go-parser", "Parser", "Name of the go struct of the generated parser"),
		goOptRemoveLib: flag.Bool("go-remove-lib", false, "Include lib in the output parser"),
	}

	flag.Parse()

	return a
}

func main() {
	var (
		a         = readArgs()
		suppress  map[int]struct{}
		errLabels map[string]string
		cfg       = langlang.NewConfig()
	)

	if *a.grammarPath == "" {
		log.Fatal("Grammar not informed")
	}

	cfg.SetBool("grammar.add_builtins", !*a.disableBuiltins)
	cfg.SetBool("grammar.add_charsets", !*a.disableCharsets)
	cfg.SetBool("grammar.captures", !*a.disableCaptures)
	cfg.SetBool("grammar.capture_spaces", !*a.disableCaptureSpaces)
	cfg.SetBool("grammar.disable_spaces", *a.disableSpaces)
	cfg.SetInt("compiler.optimize", 1)

	ast, err := langlang.GrammarFromFile(*a.grammarPath, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if *a.grammarAST {
		fmt.Println(ast.HighlightPrettyString())
		return
	}

	// Translate the AST into bytecode

	asm, err := langlang.Compile(ast, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if *a.grammarASM {
		fmt.Println(asm.HighlightPrettyString())
		return
	}

	if *a.suppressSpaces {
		suppress = map[int]struct{}{asm.StringID("Spacing"): struct{}{}}
	}

	// If it's interactive, it will open a lil REPL shell

	if *a.inputPath == "" && *a.outputPath == "" {
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

			input := langlang.NewMemInput(text)
			val, _, err := code.MatchE(&input, errLabels, suppress, *a.showFails, 0)
			if err != nil {
				fmt.Println("ERROR: " + err.Error())
			} else if val != nil {
				fmt.Println(val.HighlightPrettyString())
			}
		}
		return
	}

	// if there's an input path but no output path, run the match
	// and output the results to the standard output

	if *a.inputPath != "" && *a.outputPath == "" {
		text, err := os.ReadFile(*a.inputPath)
		if err != nil {
			log.Fatalf("Can't open input file: %s", err.Error())
		}
		code := langlang.Encode(asm)
		input := langlang.NewMemInput(string(text))
		val, _, err := code.MatchE(&input, errLabels, suppress, *a.showFails, 0)
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
		outputData, err = langlang.GenGoEval(asm, langlang.GenGoOptions{
			PackageName: *a.goOptPackage,
			ParserName:  *a.goOptParser,
			RemoveLib:   *a.goOptRemoveLib,
		})

	case "go_legacy":
		outputData, err = langlang.GenGo(ast, langlang.GenGoOptions{
			PackageName: *a.goOptPackage,
			ParserName:  *a.goOptParser,
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
