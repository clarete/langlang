package parser

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ../../../grammars/langlang.peg -output-language go -output-path ./langlang.go
//go:generate go run ../../cmd/langlang -grammar ../../../grammars/langlang.peg -output-language go -output-path ./langlang.nocap.go -disable-captures -go-parser NoCapParser -go-remove-lib

var grammarNames = []string{"csv", "json", "peg", "langlang"}

func getGrammars(tb testing.TB) map[string][]byte {
	tb.Helper()

	grammars := make(map[string][]byte, len(grammarNames))
	read := func(n string) []byte {
		text, err := os.ReadFile(fmt.Sprintf("../../../grammars/%s.peg", n))
		require.NoError(tb, err)
		return text
	}
	for _, name := range grammarNames {
		grammars[name] = read(name)
	}
	return grammars
}

func BenchmarkParser(b *testing.B) {
	grammars := getGrammars(b)
	p := NewParser()
	p.SetShowFails(false)

	for _, name := range grammarNames {
		b.Run(fmt.Sprintf("Grammar %s", name), func(b *testing.B) {
			input := grammars[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				p.ParseGrammar()
			}
		})
	}
}

func BenchmarkNoCapParser(b *testing.B) {
	grammars := getGrammars(b)
	p := NewNoCapParser()
	p.SetShowFails(false)

	for _, name := range grammarNames {
		b.Run(fmt.Sprintf("Grammar %s", name), func(b *testing.B) {
			input := grammars[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				p.ParseGrammar()
			}
		})
	}
}
