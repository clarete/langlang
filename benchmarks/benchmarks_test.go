package benchmarks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/clarete/langlang/go"
	_ "github.com/mna/pigeon/builder"
	_ "github.com/pointlander/peg/set"

	"github.com/antlr4-go/antlr/v4"
	"github.com/buger/jsonparser"
	antlr_json "github.com/clarete/langlang/benchmarks/antlr_json"
	pigeon_json "github.com/clarete/langlang/benchmarks/pigeon_json"
	pl_full "github.com/clarete/langlang/benchmarks/pointlander_json"
	pl_stripped "github.com/clarete/langlang/benchmarks/pointlander_json_stripped"
	ts_json "github.com/clarete/langlang/benchmarks/treesitter_json"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:generate go run ../go/cmd/langlang -grammar ../grammars/json.peg -output-language go -output-path ./json.go -disable-capture-spaces -go-package benchmarks
//go:generate go run ../go/cmd/langlang -grammar ../grammars/json.peg -output-language go -output-path ./json.nocap.go -disable-captures -go-package benchmarks -go-parser NoCapParser -go-remove-lib
//go:generate go run ../go/cmd/langlang -grammar ../grammars/json.stripped.peg -output-language go -go-remove-lib -go-package benchmarks -output-path ./json.stripped.go -disable-capture-spaces -go-parser StrippedParser
//go:generate go run ../go/cmd/langlang -grammar ../grammars/json.stripped.peg -output-language go -go-remove-lib -go-package benchmarks -output-path ./json.stripped.nocap.go -disable-captures -go-parser StrippedNoCapParser
//go:generate pigeon -o ./pigeon_json/json.pigeon.go ./pigeon_json/json.pigeon.peg
//go:generate peg -inline ./pointlander_json/json.pointlander.peg
//go:generate peg -inline ./pointlander_json_stripped/json.stripped.pointlander.peg
//go:generate java -jar antlr-4.13.1-complete.jar -Dlanguage=Go -o . -package antlr_json ./antlr_json/JSON.g4

// BenchmarkParsers compares JSON parsing performance across libraries.
//
// Parser lifecycle varies by library and reflects intended real-world usage:
//   - langlang, treesitter: parser created once and reused across iterations
//   - pigeon, pointlander, antlr: parser created fresh each iteration
//   - encoding_json: stateless function, no parser object
//   - buger_jsonparser: streaming/callback-based, iterates values without building a tree
func BenchmarkParsers(b *testing.B) {
	inputs := []struct {
		name string
		file string
	}{
		{"30kb", "input_30kb.json"},
		{"500kb", "input_500kb.json"},
		{"2000kb", "input_2000kb.json"},
	}

	parsers := []struct {
		name string
		fn   func(*testing.B, []byte)
	}{
		{"encoding_json", benchmarkEncodingJSON},
		{"buger_jsonparser", benchmarkBugerJSONParser},
		{"pigeon", benchmarkPigeonParser},
		{"pointlander", benchmarkPointlanderParser},
		{"pointlander_stripped", benchmarkPointlanderStrippedParser},
		{"antlr", benchmarkAntlrParser},
		{"langlang", benchmarkLanglangParser},
		{"langlang_nocap", benchmarkLanglangNoCapParser},
		{"langlang_stripped", benchmarkLanglangStrippedParser},
		{"langlang_stripped_nocap", benchmarkLanglangStrippedNoCapParser},
		{"treesitter", benchmarkTreeSitterParser},
	}

	for _, input := range inputs {
		data := loadInput(b, input.file)
		for _, parser := range parsers {
			version := getVersion(parser.name)
			name := fmt.Sprintf("input=%s/parser=%s/version=%s", input.name, parser.name, version)
			fn := parser.fn
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(len(data)))
				fn(b, data)
			})
		}
	}
}

func loadInput(tb testing.TB, filename string) []byte {
	tb.Helper()
	path := filepath.Join(".", "input", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("failed to read input file %s: %v", path, err)
	}
	return data
}

func benchmarkEncodingJSON(b *testing.B, data []byte) {
	var v any
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := json.Unmarshal(data, &v); err != nil {
			b.Fatalf("error in encoding/json: %v", err)
		}
	}
}

func benchmarkBugerJSONParser(b *testing.B, data []byte) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			jsonparser.ObjectEach(value, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
				return nil
			})
		})
		if err != nil {
			b.Fatalf("error in buger/jsonparser: %v", err)
		}
	}
}

func benchmarkPigeonParser(b *testing.B, data []byte) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pigeon_json.Parse("", data)
		if err != nil {
			b.Fatalf("error in pigeon parser: %v", err)
		}
	}
}

func benchmarkPointlanderParser(b *testing.B, data []byte) {
	dataStr := string(data)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := &pl_full.PointlanderJSON{
			Buffer: dataStr,
		}
		if err := p.Init(); err != nil {
			b.Fatalf("error in pointlander init: %v", err)
		}
		if err := p.Parse(); err != nil {
			b.Fatalf("error in pointlander parse: %v", err)
		}
	}
}

func benchmarkPointlanderStrippedParser(b *testing.B, data []byte) {
	dataStr := string(data)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := &pl_stripped.PointlanderStrippedJSON{
			Buffer: dataStr,
		}
		if err := p.Init(); err != nil {
			b.Fatalf("error in pointlander_stripped init: %v", err)
		}
		if err := p.Parse(); err != nil {
			b.Fatalf("error in pointlander_stripped parse: %v", err)
		}
	}
}

func benchmarkLanglangParser(b *testing.B, data []byte) {
	p := NewParser()
	p.SetInput(data)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := p.ParseJSON()
		if err != nil {
			b.Fatalf("error in PEG parser: %v", err)
		}
	}
}

func benchmarkLanglangNoCapParser(b *testing.B, data []byte) {
	p := NewNoCapParser()
	p.SetInput(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.ParseJSON()
		if err != nil {
			b.Fatalf("error in PEG parser: %v", err)
		}
	}
}

func benchmarkLanglangStrippedParser(b *testing.B, data []byte) {
	p := NewStrippedParser()
	p.SetInput(data)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := p.ParseJSON()
		if err != nil {
			b.Fatalf("error in stripped PEG parser: %v", err)
		}
	}
}

func benchmarkLanglangStrippedNoCapParser(b *testing.B, data []byte) {
	p := NewStrippedNoCapParser()
	p.SetInput(data)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := p.ParseJSON()
		if err != nil {
			b.Fatalf("error in stripped no cap PEG parser: %v", err)
		}
	}
}

func benchmarkAntlrParser(b *testing.B, data []byte) {
	dataStr := string(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := antlr.NewInputStream(dataStr)
		lexer := antlr_json.NewJSONLexer(input)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := antlr_json.NewJSONParser(stream)
		parser.BuildParseTrees = true
		_ = parser.Json()
	}
}

func benchmarkTreeSitterParser(b *testing.B, data []byte) {
	parser := sitter.NewParser()
	defer parser.Close()

	lang := ts_json.GetJSONLanguage()
	parser.SetLanguage(lang)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree := parser.Parse(data, nil)
		if tree == nil {
			b.Fatal("failed to parse with tree-sitter")
		}
		tree.Close()
	}
}

var moduleVersions = parseGoMod()

func getVersion(parser string) string {
	switch parser {
	case "encoding_json":
		// stdlib version is Go version
		return runtime.Version()
	case "langlang", "langlang_nocap", "langlang_stripped", "langlang_stripped_nocap":
		// Use env var set by run script, fallback to "dev"
		if v := os.Getenv("LANGLANG_VERSION"); v != "" {
			return v
		}
		return "dev"
	case "buger_jsonparser":
		if v, ok := moduleVersions["github.com/buger/jsonparser"]; ok {
			return v
		}
	case "pigeon":
		if v, ok := moduleVersions["github.com/mna/pigeon"]; ok {
			return v
		}
	case "pointlander", "pointlander_stripped":
		if v, ok := moduleVersions["github.com/pointlander/peg"]; ok {
			return v
		}
	case "antlr":
		if v, ok := moduleVersions["github.com/antlr4-go/antlr/v4"]; ok {
			return v
		}
	case "treesitter":
		if v, ok := moduleVersions["github.com/tree-sitter/go-tree-sitter"]; ok {
			return v
		}
	}
	return "unknown"
}

func parseGoMod() map[string]string {
	versions := make(map[string]string)

	f, err := os.Open("go.mod")
	if err != nil {
		return versions
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Match lines like: github.com/buger/jsonparser v1.1.1
		if strings.HasPrefix(line, "github.com/") || strings.HasPrefix(line, "golang.org/") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				module := parts[0]
				version := strings.TrimSuffix(parts[1], "//")
				version = strings.TrimSpace(version)
				versions[module] = version
			}
		}
	}
	return versions
}
