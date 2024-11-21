package main

import (
	"flag"
	"log"
	"os"

	langlang "github.com/clarete/langlang/go"
)

const defaultWritePermission = 0644 // -rw-r--r--

func main() {
	var (
		grammarPath = flag.String("grammar", "", "Path to the grammar file")
		outputPath  = flag.String("output", "/dev/stdout", "Path to the output file")
		language    = flag.String("language", "", "Output language")
		astOnly     = flag.Bool("ast-only", false, "Output the AST of the grammar")

		// removeLib will configure the generator to not
		// include the common prelude in the output parser.
		removeLib = flag.Bool("remove-lib", false, "Include lib in the output parser")

		// goOptPackage
		goOptPackage = flag.String("go-package", "parser", "Name of the go package in the generated parser")
	)
	flag.Parse()

	if *language == "" {
		*language = "go"
	}

	if *grammarPath == "" {
		log.Fatal("Grammar not informed")
	}

	importLoader := langlang.NewRelativeImportLoader()
	importResolver := langlang.NewImportResolver(importLoader)
	ast, err := importResolver.Resolve(*grammarPath)
	if err != nil {
		log.Fatal(err)
	}
	if *astOnly {
		log.Println(ast)
		return
	}

	var outputData string
	switch *language {
	case "go":
		outputData, err = langlang.GenGo(ast, langlang.GenGoOptions{
			PackageName: *goOptPackage,
			RemoveLib:   *removeLib,
		})

	case "python":
		outputData, err = langlang.GenPy(ast, langlang.GenPyOptions{
			GrammarPath: *grammarPath,
			RemoveLib:   *removeLib,
		})
	case "typescript":
		outputData, err = langlang.GenTs(ast, langlang.GenTsOptions{
			GrammarPath: *grammarPath,
			RemoveLib:   *removeLib,
		})
	case "javascript":
		outputData, err = langlang.GenJs(ast, langlang.GenJsOptions{
			GrammarPath: *grammarPath,
			RemoveLib:   *removeLib,
		})
	default:
		log.Fatalf("Output language `%s` not supported", *language)
	}
	if err != nil {
		log.Fatalf("Can't emit code: %s", err.Error())
	}

	if err = os.WriteFile(*outputPath, []byte(outputData), 0644); err != nil {
		log.Fatalf("Can't write output: %s", err.Error())
	}
}
