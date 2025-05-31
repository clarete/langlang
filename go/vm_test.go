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

		vm := NewVirtualMachine(bytecode)

		_, cur, err := vm.Match(strings.NewReader(""))

		require.NoError(t, err)
		assert.Equal(t, 0, cur)
	})

	t.Run("did the cursor move", func(t *testing.T) {
		_, cur, err := exec("G <- .", "foo", 1)
		require.NoError(t, err)
		assert.Equal(t, 1, cur)
	})

	vmTests := []struct {
		Name           string
		Grammar        string
		Input          string
		ExpectedCursor int
		ExpectedAST    *Node
	}{
		{
			Name:           "Any",
			Grammar:        "G <- .",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST:    NewNode("G", NewString("f", sp(1)), sp(1)),
		},
		{
			Name:           "Any Star",
			Grammar:        "G <- .*",
			Input:          "foo",
			ExpectedCursor: 3,
			ExpectedAST:    NewNode("G", NewString("foo", sp(3)), sp(3)),
		},
		{
			Name:           "Char",
			Grammar:        "G <- 'f'",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST:    NewNode("G", NewString("f", sp(1)), sp(1)),
		},
		{
			Name:           "Choice",
			Grammar:        "G <- 'f' / 'g' / 'h'",
			Input:          "g",
			ExpectedCursor: 1,
			ExpectedAST:    NewNode("G", NewString("g", sp(1)), sp(1)),
		},
		{
			Name:           "Choice on Words",
			Grammar:        "G <- 'avocado' / 'avante' / 'aviador'",
			Input:          "avante",
			ExpectedCursor: 6,
			ExpectedAST:    NewNode("G", NewString("avante", sp(6)), sp(6)),
		},
		{
			Name:           "Class with Range",
			Grammar:        "G <- [0-9]+",
			Input:          "42",
			ExpectedCursor: 2,
			ExpectedAST:    NewNode("G", NewString("42", sp(2)), sp(2)),
		},
		{
			Name:           "Class with Range and Literal",
			Grammar:        "G <- [a-z_]+",
			Input:          "my_id",
			ExpectedCursor: 5,
			ExpectedAST:    NewNode("G", NewString("my_id", sp(5)), sp(5)),
		},
		{
			Name:           "Optional Matches",
			Grammar:        "G <- 'f'?",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST:    NewNode("G", NewString("f", sp(1)), sp(1)),
		},
		{
			Name:           "Optional does not match",
			Grammar:        "G <- 'f'?",
			Input:          "bar",
			ExpectedCursor: 0,
			ExpectedAST:    nil,
		},
		{
			Name:           "Optional does not match followed by something else",
			Grammar:        "G <- 'f'? 'bar'",
			Input:          "bar",
			ExpectedCursor: 3,
			ExpectedAST:    NewNode("G", NewString("bar", sp(3)), sp(3)),
		},
		{
			Name:           "Not predicate",
			Grammar:        "G <- (!';' .)*",
			Input:          "foo; bar",
			ExpectedCursor: 3,
			ExpectedAST:    NewNode("G", NewString("foo", sp(3)), sp(3)),
		},
		{
			Name:           "Not Any and Star",
			Grammar:        `G <- "'" (!"'" .)* "'"`,
			Input:          "'foo'",
			ExpectedCursor: 5,
			ExpectedAST:    NewNode("G", NewString("'foo'", sp(5)), sp(5)),
		},
		{
			Name:           "And predicate",
			Grammar:        "G <- &'a' .",
			Input:          "avocado",
			ExpectedCursor: 1,
			ExpectedAST:    NewNode("G", NewString("a", sp(1)), sp(1)),
		},
		{
			Name:           "Parse HEX Number",
			Grammar:        "G <- '0x' [0-9a-fA-F]+ / '0'",
			Input:          "0xff",
			ExpectedCursor: 4,
			ExpectedAST:    NewNode("G", NewString("0xff", sp(4)), sp(4)),
		},
		{
			Name:           "Unicode",
			Grammar:        "G <- [♡]",
			Input:          "♡",
			ExpectedCursor: 3,
			ExpectedAST:    NewNode("G", NewString("♡", sp(3)), sp(3)),
		},
	}

	for _, test := range vmTests {
		t.Run("O0 "+test.Name, func(t *testing.T) {
			val, cur, err := exec(test.Grammar, test.Input, 0)
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedCursor, cur)
			if test.ExpectedAST == nil {
				assert.Nil(t, val)
			} else {
				assert.Equal(t, test.ExpectedAST, val)
			}
		})

		t.Run("O1 "+test.Name, func(t *testing.T) {
			val, cur, err := exec(test.Grammar, test.Input, 1)
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedCursor, cur)
			if test.ExpectedAST == nil {
				assert.Nil(t, val)
			} else {
				assert.Equal(t, test.ExpectedAST, val)
			}
		})
	}
}

func sp(ch int) Span {
	return NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, ch))
}

func exec(expr, input string, optimize int) (Value, int, error) {
	ast, err := NewGrammarParser(expr).Parse()
	if err != nil {
		panic(err)
	}
	ast, err = AddCaptures(ast)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ast\n%s\n", ast.PrettyPrint())
	asm, err := Compile(ast, CompilerConfig{Optimize: optimize})
	if err != nil {
		panic(err)
	}
	fmt.Printf("asm\n%s\n", asm.PrettyPrint())

	code := Encode(asm)

	fmt.Printf("code\n%#v\n", code.code)

	return code.Match(strings.NewReader(input))
}
