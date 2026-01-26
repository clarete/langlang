package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/clarete/langlang/go"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[1;31m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[1;36m"
	colorGray   = "\033[0;37m"
)

type args struct {
	grammarPath *string

	grammarAST *bool
	grammarASM *bool
	grammarMAP *bool

	outputLang *string
	outputPath *string

	disableBuiltins      *bool
	disableSpaces        *bool
	disableCharsets      *bool
	disableCaptures      *bool
	disableCaptureSpaces *bool
	disableInline        *bool
	disableInlineDefs    *bool

	showFails *bool

	inputPath   *string
	interactive *bool

	goOptPackage   *string
	goOptParser    *string
	goOptRemoveLib *bool

	diagnosticLevel *string
}

func readArgs() *args {
	a := &args{
		grammarPath: flag.String("grammar", "", "Path to the grammar file"),

		// Debugging Options

		grammarAST: flag.Bool("grammar-ast", false, "Output the AST of the grammar"),
		grammarASM: flag.Bool("grammar-asm", false, "Output the ASM of the grammar"),
		grammarMAP: flag.Bool("grammar-source-map", false, "Include the source map of the grammar in the output parser"),

		// Output Options

		disableBuiltins:      flag.Bool("disable-builtins", false, "Tells the compiler not to inject builtin rules into the grammar"),
		disableSpaces:        flag.Bool("disable-spaces", false, "Tells the compiler not to inject automatic whitespace char handling into the grammar"),
		disableCharsets:      flag.Bool("disable-charsets", false, "Inject whitespace handling rules into the grammar"),
		disableCaptures:      flag.Bool("disable-captures", false, "Tells the compiler not to inject capture rules into the grammar"),
		disableCaptureSpaces: flag.Bool("disable-capture-spaces", false, "Tells the compiler not to inject capture rules for spaces into the grammar"),
		disableInline:        flag.Bool("disable-inline", false, "Tells the compiler not to inline any definitions"),
		disableInlineDefs:    flag.Bool("disable-inline-defs", true, "Tells the compiler not to emit Parse methods for inlined definitions (saves space)"),
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

		diagnosticLevel: flag.String("diagnostics", "error", "Minimum diagnostic level to display: error, warning, info, hint, or all"),
	}

	flag.Parse()

	return a
}

func main() {
	a := readArgs()

	if *a.grammarPath == "" {
		fatal("Grammar not informed")
	}

	cfg := langlang.NewConfig()
	cfg.SetBool("grammar.add_builtins", !*a.disableBuiltins)
	cfg.SetBool("grammar.add_charsets", !*a.disableCharsets)
	cfg.SetBool("grammar.captures", !*a.disableCaptures)
	cfg.SetBool("grammar.capture_spaces", !*a.disableCaptureSpaces)
	cfg.SetBool("grammar.handle_spaces", !*a.disableSpaces)
	cfg.SetBool("compiler.inline.enabled", !*a.disableInline)
	cfg.SetBool("compiler.inline.emit.inlined", !*a.disableInlineDefs)
	cfg.SetBool("vm.show_fails", *a.showFails)
	cfg.SetBool("vm.debug.source_map", *a.grammarMAP)

	// Create a query-based database for caching and diagnostics
	loader := langlang.NewRelativeImportLoader()
	db := langlang.NewDatabase(cfg, loader)

	// Parse the diagnostic level
	minLevel := parseDiagnosticLevel(*a.diagnosticLevel)

	// Check for diagnostics (parse errors, semantic errors) first
	diagnostics, err := langlang.QueryDiagnostics(db, *a.grammarPath)
	if err != nil {
		// Check if it's a GrammarError with diagnostics we can display
		if grammarErr, ok := err.(*langlang.GrammarError); ok {
			printDiagnosticsAndCheckForErrors(grammarErr.Diagnostics, minLevel)
			os.Exit(1)
		}
		fatal("Failed to analyze grammar: %s", err.Error())
	}

	// If there are errors, exit early (warnings/info are okay)
	if printDiagnosticsAndCheckForErrors(diagnostics, minLevel) {
		os.Exit(1)
	}

	// Get the AST using the query system
	ast, err := langlang.QueryAST(db, *a.grammarPath)
	if err != nil {
		fatal("Failed to parse grammar: %s", err.Error())
	}

	if *a.grammarAST {
		fmt.Println(ast.HighlightPrettyString())
		return
	}

	// Get the compiled program using the query system
	program, err := langlang.QueryProgram(db, *a.grammarPath)
	if err != nil {
		fatal("Failed to compile grammar: %s", err.Error())
	}

	if *a.grammarASM {
		fmt.Println(program.HighlightPrettyString())
		return
	}

	// If it's interactive, it will open a lil REPL shell
	if *a.inputPath == "" && *a.outputPath == "" {
		matcher, err := langlang.QueryMatcher(db, *a.grammarPath)
		if err != nil {
			fatal("Failed to create matcher: %s", err.Error())
		}

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

			input := []byte(text)
			tree, _, err := matcher.Match(input)
			if err != nil {
				printParsingError(err, matcher)
			} else if tree != nil {
				root, _ := tree.Root()
				fmt.Println(tree.Highlight(root))
			}
		}
		return
	}

	// if there's an input path but no output path, run the match
	// and output the results to the standard output
	if *a.inputPath != "" && *a.outputPath == "" {
		text, err := os.ReadFile(*a.inputPath)
		if err != nil {
			fatal("Can't open input file: %s", err.Error())
		}
		matcher, err := langlang.QueryMatcher(db, *a.grammarPath)
		if err != nil {
			fatal("Failed to create matcher: %s", err.Error())
		}
		tree, _, err := matcher.Match(text)
		if err != nil {
			printParsingError(err, matcher)
		} else if tree != nil {
			root, _ := tree.Root()
			fmt.Println(tree.Highlight(root))
		}
		return
	}

	// Configure output options
	if *a.outputLang == "" {
		fatal("Expected `-output-language`")
	}

	var outputData string
	switch *a.outputLang {
	case "go":
		outputData, err = langlang.GenGoEval(program, cfg, langlang.GenGoOptions{
			PackageName: *a.goOptPackage,
			ParserName:  *a.goOptParser,
			RemoveLib:   *a.goOptRemoveLib,
		})

	// case "python":
	// 	outputData, err = langlang.GenParserPython(ast)
	default:
		fatal("Output language `%s` not supported", *a.outputLang)
	}
	if err != nil {
		fatal("Can't emit code: %s", err.Error())
	}

	if err = os.WriteFile(*a.outputPath, []byte(outputData), 0644); err != nil {
		fatal("Can't write output: %s", err.Error())
	}
}

// parseDiagnosticLevel converts a string level to DiagnosticSeverity.
// Returns the severity and whether to include all levels below it.
func parseDiagnosticLevel(level string) langlang.DiagnosticSeverity {
	switch level {
	case "error":
		return langlang.DiagnosticError
	case "warning":
		return langlang.DiagnosticWarning
	case "info":
		return langlang.DiagnosticInfo
	case "hint", "all":
		return langlang.DiagnosticHint
	default:
		return langlang.DiagnosticError
	}
}

func printDiagnosticsAndCheckForErrors(diagnostics []langlang.Diagnostic, minLevel langlang.DiagnosticSeverity) bool {
	hasErrors := false
	errorCount := 0
	warningCount := 0
	infoCount := 0
	hintCount := 0

	for _, d := range diagnostics {
		// Count all diagnostics regardless of filter
		switch d.Severity {
		case langlang.DiagnosticError:
			errorCount++
		case langlang.DiagnosticWarning:
			warningCount++
		case langlang.DiagnosticInfo:
			infoCount++
		case langlang.DiagnosticHint:
			hintCount++
		}

		// Filter based on minimum level (lower value = higher severity)
		if d.Severity > minLevel {
			continue
		}

		loc := d.Location.Span.Start

		// Choose color based on severity
		var severityColor string
		switch d.Severity {
		case langlang.DiagnosticError:
			severityColor = colorRed
			hasErrors = true
		case langlang.DiagnosticWarning:
			severityColor = colorYellow
		case langlang.DiagnosticInfo:
			severityColor = colorCyan
		case langlang.DiagnosticHint:
			severityColor = colorGray
		}

		fmt.Printf("%s%s:%d:%d:%s %s%s:%s %s %s[%s]%s\n",
			colorGray, d.FilePath, loc.Line, loc.Column, colorReset,
			severityColor, d.Severity, colorReset, d.Message,
			colorGray, d.Code, colorReset)
	}

	// Print summary if there are any diagnostics at or above the filter level
	printedCount := 0
	for _, d := range diagnostics {
		if d.Severity <= minLevel {
			printedCount++
		}
	}

	if printedCount > 0 {
		fmt.Printf("\n")
		parts := []string{}
		if errorCount > 0 && minLevel >= langlang.DiagnosticError {
			parts = append(parts, fmt.Sprintf("%s%d error(s)%s", colorRed, errorCount, colorReset))
		}
		if warningCount > 0 && minLevel >= langlang.DiagnosticWarning {
			parts = append(parts, fmt.Sprintf("%s%d warning(s)%s", colorYellow, warningCount, colorReset))
		}
		if infoCount > 0 && minLevel >= langlang.DiagnosticInfo {
			parts = append(parts, fmt.Sprintf("%s%d info%s", colorCyan, infoCount, colorReset))
		}
		if hintCount > 0 && minLevel >= langlang.DiagnosticHint {
			parts = append(parts, fmt.Sprintf("%s%d hint(s)%s", colorGray, hintCount, colorReset))
		}
		if len(parts) > 0 {
			fmt.Printf("%s generated\n", strings.Join(parts, ", "))
		}
	}

	return hasErrors
}

// printParsingError prints a parsing error with grammar location if available.
func printParsingError(err error, matcher langlang.Matcher) {
	perr, ok := err.(langlang.ParsingError)
	if !ok {
		fmt.Printf("%sERROR:%s %s\n", colorRed, colorReset, err.Error())
		return
	}

	fmt.Printf("%sERROR:%s %s\n", colorRed, colorReset, perr.Message)

	// Show grammar location if source map is available
	if srcm := matcher.SourceMap(); srcm != nil {
		if loc, ok := srcm.LocationAt(perr.FFPPC); ok {
			file := srcm.FileAt(loc.FileID)
			fmt.Printf(" %s at %s:%d:%d%s\n",
				colorGray,
				file,
				loc.Span.Start.Line,
				loc.Span.Start.Column,
				colorReset,
			)
		}
	}
}

// fatal prints an error message and exits with code 1.
func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%serror:%s ", colorRed, colorReset)
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
