package langlang

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVM(t *testing.T) {
	t.Run("I guess I will just die", func(t *testing.T) {
		bytecode := Encode(&Program{code: []Instruction{
			IHalt{},
		}})
		assert.Equal(t, uint8(0), bytecode.code[0])

		_, cur, err := bytecode.Match(strings.NewReader(""))

		require.NoError(t, err)
		assert.Equal(t, 0, cur)
	})

	t.Run("one day", func(t *testing.T) {
		/* val */ _, cur, err := run("G <- .", "foo")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)
	})
}

func run(expr, input string) (Value, int, error) {
	ast, err := NewGrammarParser(expr).Parse()
	if err != nil {
		panic(err)
	}
	fmt.Printf("ast\n%s\n", ast.PrettyPrint())

	asm, err := Compile(ast, CompilerConfig{Optimize: 1})
	if err != nil {
		panic(err)
	}

	fmt.Printf("asm\n%s\n", asm.PrettyPrint())
	bytecode := Encode(asm)
	return bytecode.Match(strings.NewReader(input))
}
