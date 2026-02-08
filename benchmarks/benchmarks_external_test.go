//go:build external

package benchmarks

import (
	"encoding/json"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/buger/jsonparser"
	antlr_json "github.com/clarete/langlang/benchmarks/antlr_json"
	pigeon_json "github.com/clarete/langlang/benchmarks/pigeon_json"
	pl_full "github.com/clarete/langlang/benchmarks/pointlander_json"
	pl_stripped "github.com/clarete/langlang/benchmarks/pointlander_json_stripped"
	ts_json "github.com/clarete/langlang/benchmarks/treesitter_json"
	_ "github.com/mna/pigeon/builder"
	_ "github.com/pointlander/peg/set"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:generate pigeon -o ./pigeon_json/json.pigeon.go ./pigeon_json/json.pigeon.peg
//go:generate peg -inline ./pointlander_json/json.pointlander.peg
//go:generate peg -inline ./pointlander_json_stripped/json.stripped.pointlander.peg
//go:generate java -jar antlr-4.13.1-complete.jar -Dlanguage=Go -o . -package antlr_json ./antlr_json/JSON.g4

func init() {
	registerParser("encoding_json", benchmarkEncodingJSON)
	registerParser("buger_jsonparser", benchmarkBugerJSONParser)
	registerParser("pigeon", benchmarkPigeonParser)
	registerParser("pointlander", benchmarkPointlanderParser)
	registerParser("pointlander_stripped", benchmarkPointlanderStrippedParser)
	registerParser("antlr", benchmarkAntlrParser)
	registerParser("treesitter", benchmarkTreeSitterParser)
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

