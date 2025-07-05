package parser

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ../../../grammars/json.peg -output-language goeval -output-path ./json.go

func BenchmarkParser(b *testing.B) {

	inputNames := []string{"30kb", "500kb", "2000kb"}

	inputs := make(map[string]string, len(inputNames))

	read := func(n string) string {
		text, err := os.ReadFile(fmt.Sprintf("./input_%s.json", n))
		require.NoError(b, err)
		return string(text)
	}

	for _, name := range inputNames {
		inputs[name] = read(name)
	}

	for _, name := range inputNames {
		b.Run(fmt.Sprintf("Input %s", name), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				p := NewParser()
				p.SetInput(inputs[name])
				p.ParseJSON()
			}
		})
	}
}
