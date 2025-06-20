package parser

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ../../../grammars/langlang.peg -output-language goeval -output-path ./langlang.go

func BenchmarkParser(b *testing.B) {

	grammarNames := []string{"csv", "json", "peg", "langlang"}

	grammars := make(map[string]string, len(grammarNames))

	read := func(n string) string {
		text, err := os.ReadFile(fmt.Sprintf("../../../grammars/%s.peg", n))
		require.NoError(b, err)
		return string(text)
	}

	for _, name := range grammarNames {
		grammars[name] = read(name)
	}

	for _, name := range grammarNames {
		b.Run(fmt.Sprintf("Grammar %s", name), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				p := NewParser()
				p.SetInput(grammars[name])
				p.ParseGrammar()
			}
		})
	}
}
