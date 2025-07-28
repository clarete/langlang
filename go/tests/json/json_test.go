package parser

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ../../../grammars/json.peg -output-language go -output-path ./json.go
//go:generate go run ../../cmd/langlang -grammar ../../../grammars/json.peg -output-language go -output-path ./json.nocap.go -disable-captures -go-parser NoCapParser -go-remove-lib
//go:generate go run ../../cmd/langlang -grammar ../../../grammars/json.stripped.peg -output-language go -go-remove-lib -output-path ./json.stripped.go -go-parser StrippedParser
//go:generate go run ../../cmd/langlang -grammar ../../../grammars/json.stripped.peg -output-language go -go-remove-lib -output-path ./json.stripped.nocap.go -disable-captures -go-parser StrippedNoCapParser

var inputNames = []string{"30kb", "500kb", "2000kb"}

func getInputs(tb testing.TB) map[string]string {
	tb.Helper()

	inputs := make(map[string]string, len(inputNames))
	read := func(n string) string {
		text, err := os.ReadFile(fmt.Sprintf("./input_%s.json", n))
		require.NoError(tb, err)
		return string(text)
	}
	for _, name := range inputNames {
		inputs[name] = read(name)
	}
	return inputs
}

func BenchmarkParser(b *testing.B) {
	inputs := getInputs(b)

	b.ResetTimer()
	p := NewParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(fmt.Sprintf("Input %s", name), func(b *testing.B) {
			p.SetInput(inputs[name])

			for n := 0; n < b.N; n++ {
				p.ParseJSON()
			}
		})
	}
}

func BenchmarkNoCapParser(b *testing.B) {
	inputs := getInputs(b)

	b.ResetTimer()
	p := NewNoCapParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(fmt.Sprintf("Input %s", name), func(b *testing.B) {
			p.SetInput(inputs[name])

			for n := 0; n < b.N; n++ {
				p.ParseJSON()
			}
		})
	}
}

func BenchmarkStrippedParser(b *testing.B) {
	inputs := getInputs(b)

	b.ResetTimer()
	p := NewStrippedParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(fmt.Sprintf("Input %s", name), func(b *testing.B) {
			p.SetInput(inputs[name])

			for n := 0; n < b.N; n++ {
				p.ParseJSON()
			}
		})
	}
}

func BenchmarkStrippedNoCapParser(b *testing.B) {
	inputs := getInputs(b)

	b.ResetTimer()
	p := NewStrippedNoCapParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(fmt.Sprintf("Input %s", name), func(b *testing.B) {
			p.SetInput(inputs[name])

			for n := 0; n < b.N; n++ {
				p.ParseJSON()
			}
		})
	}
}
