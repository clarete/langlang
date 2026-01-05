// vmdbgen generates VM variants from the MatchRule function in vm.go.
//
// Usage:
//
//	go run ./vmdbgen -mode debug -vm-file ../vm.go -output-path ../vm_debug.go
//	go run ./vmdbgen -mode oracle -vm-file ../vm.go -output-path ../vm_oracle.go
//
// The generator parses MatchRule and applies transformations based on
// the selected flavor (mode), producing a new function with different
// behavior while reusing the carefully optimized VM loop.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
)

var flavors = map[string]VMFlavor{
	"debug":  DebugFlavor,
	"oracle": OracleFlavor,
}

func main() {
	var (
		vmFile     = flag.String("vm-file", "../vm.go", "path to the VM source file")
		outputPath = flag.String("output-path", "", "output path (default: flavor's OutputFile)")
		mode       = flag.String("mode", "debug", "flavor mode: debug, oracle")
	)
	flag.Parse()

	flavor, ok := flavors[*mode]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown mode: %s\nAvailable modes: ", *mode)
		for name := range flavors {
			fmt.Fprintf(os.Stderr, "%s ", name)
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	outPath := *outputPath
	if outPath == "" {
		outPath = flavor.OutputFile
	}

	fset := token.NewFileSet()
	data, err := os.ReadFile(*vmFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read vm file: %v", err))
	}

	node, err := parser.ParseFile(fset, *vmFile, data, parser.AllErrors)
	if err != nil {
		panic(fmt.Sprintf("failed to parse vm file: %v", err))
	}

	fn := getMatchRuleFunction(node)
	removeInitializationsFromFunction(fn)
	removeVarDeclarationsFromFunction(fn)
	removeRuleAddressCheckFromFunction(fn)
	adjustFunctionSignature(fn, flavor)
	if flavor.Features.EOFReturnsSuccess {
		transformEOFToSuccess(fn, flavor.MkEOFReturn)
	}
	if flavor.Features.SpanStaysAtEOF {
		transformSpanForOracle(fn, flavor.MkEOFReturn)
	}
	addHooksToFunction(fn, flavor)
	adjustReturnStatements(fn, flavor)
	for _, transform := range flavor.ExtraTransforms {
		transform(fn)
	}

	var output strings.Builder
	pkg := mkPackage(fn)
	if err := printer.Fprint(&output, fset, pkg); err != nil {
		panic(fmt.Sprintf("failed to print AST: %v", err))
	}
	formatted, err := format.Source([]byte(output.String()))
	if err != nil {
		panic(fmt.Sprintf("failed to format output: %v", err))
	}
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		panic(fmt.Sprintf("failed to write output: %v", err))
	}
}
