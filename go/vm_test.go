package langlang

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vmTest struct {
	Name           string
	Grammar        string
	Input          string
	ExpectedAST    string
	ExpectedError  string
	ExpectedCursor int
}

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

	vmTests := []vmTest{
		{
			Name:           "Any",
			Grammar:        "G <- .",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST: `G (0..1)
└── "f" (0..1)`,
		},
		{
			Name:           "Any Star",
			Grammar:        "G <- .*",
			Input:          "foo",
			ExpectedCursor: 3,
			ExpectedAST: `G (0..3)
└── "foo" (0..3)`,
		},
		{
			Name:           "Char",
			Grammar:        "G <- 'f'",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST: `G (0..1)
└── "f" (0..1)`,
		},
		{
			Name:           "Choice",
			Grammar:        "G <- 'f' / 'g' / 'h'",
			Input:          "g",
			ExpectedCursor: 1,
			ExpectedAST: `G (0..1)
└── "g" (0..1)`,
		},
		{
			Name:           "Choice on Words",
			Grammar:        "G <- 'avocado' / 'avante' / 'aviador'",
			Input:          "avante",
			ExpectedCursor: 6,
			ExpectedAST: `G (0..6)
└── "avante" (0..6)`,
		},
		{
			Name:           "Class with Range",
			Grammar:        "G <- [0-9]+",
			Input:          "42",
			ExpectedCursor: 2,
			ExpectedAST: `G (0..2)
└── "42" (0..2)`,
		},
		{
			Name:           "Class with Range and Literal",
			Grammar:        "G <- [a-z_]+",
			Input:          "my_id",
			ExpectedCursor: 5,
			ExpectedAST: `G (0..5)
└── "my_id" (0..5)`,
		},
		{
			Name:           "Optional Matches",
			Grammar:        "G <- 'f'?",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST: `G (0..1)
└── "f" (0..1)`,
		},
		{
			Name:           "Optional does not match",
			Grammar:        "G <- 'f'?",
			Input:          "bar",
			ExpectedCursor: 0,
			ExpectedAST:    "",
		},
		{
			Name:           "Optional does not match followed by something else",
			Grammar:        "G <- 'f'? 'bar'",
			Input:          "bar",
			ExpectedCursor: 3,
			ExpectedAST: `G (0..3)
└── "bar" (0..3)`,
		},
		{
			Name:           "Not predicate",
			Grammar:        "G <- (!';' .)*",
			Input:          "foo; bar",
			ExpectedCursor: 3,
			ExpectedAST: `G (0..3)
└── "foo" (0..3)`,
		},
		{
			Name:           "Not Any and Star",
			Grammar:        `G <- "'" (!"'" .)* "'"`,
			Input:          "'foo'",
			ExpectedCursor: 5,
			ExpectedAST: `G (0..5)
└── "'foo'" (0..5)`,
		},
		{
			Name:           "And predicate",
			Grammar:        "G <- &'a' .",
			Input:          "avocado",
			ExpectedCursor: 1,
			ExpectedAST: `G (0..1)
└── "a" (0..1)`,
		},
		{
			Name:           "Parse HEX Number",
			Grammar:        "G <- '0x' [0-9a-fA-F]+ / '0'",
			Input:          "0xff",
			ExpectedCursor: 4,
			ExpectedAST: `G (0..4)
└── "0xff" (0..4)`,
		},
		{
			Name:           "Unicode",
			Grammar:        "G <- [♡]",
			Input:          "♡",
			ExpectedCursor: 3,
			ExpectedAST: `G (0..1)
└── "♡" (0..1)`,
		},
		{
			Name: "Var",
			Grammar: `G <- D
		D <- [0-9]+`,
			Input:          "1",
			ExpectedCursor: 1,
			ExpectedAST: `G (0..1)
└── D (0..1)
    └── "1" (0..1)`,
		},
		{
			Name: "Var and Var",
			Grammar: `G <- D P
				  D <- [0-9]+
				  P <- '!'`,
			Input:          "42!",
			ExpectedCursor: 3,
			ExpectedAST: `G (0..3)
└── Sequence<2> (0..3)
    ├── D (0..2)
    │   └── "42" (0..2)
    └── P (2..3)
        └── "!" (2..3)`,
		},
		{
			Name: "Var Char Var",
			Grammar: `G <- D '+' D
				  D <- [0-9]+`,
			Input:          "40+2",
			ExpectedCursor: 4,
			ExpectedAST: `G (0..4)
└── Sequence<3> (0..4)
    ├── D (0..2)
    │   └── "40" (0..2)
    ├── "+" (2..3)
    └── D (3..4)
        └── "2" (3..4)`,
		},
		{
			Name: "Var Char Var Char",
			Grammar: `G <- D '+' D '!'
				  D <- [0-9]+`,
			Input:          "40+2!",
			ExpectedCursor: 5,
			ExpectedAST: `G (0..5)
└── Sequence<4> (0..5)
    ├── D (0..2)
    │   └── "40" (0..2)
    ├── "+" (2..3)
    ├── D (3..4)
    │   └── "2" (3..4)
    └── "!" (4..5)`,
		},
		{
			Name: "Char Var Char",
			Grammar: `G <- '+' D '+'
				  D <- [0-9]+`,
			Input:          "+42+",
			ExpectedCursor: 4,
			ExpectedAST: `G (0..4)
└── Sequence<3> (0..4)
    ├── "+" (0..1)
    ├── D (1..3)
    │   └── "42" (1..3)
    └── "+" (3..4)`,
		},
		{
			Name: "Lexification", // TODO: Needs test for failure
			Grammar: `
                           Ordinal <- Decimal #('st' / 'nd' / 'rd' / 'th')
                           Decimal <- ([1-9][0-9]*) / '0'
                        `,
			Input:          "42nd",
			ExpectedCursor: 4,
			ExpectedAST: `Ordinal (0..4)
└── Sequence<2> (0..4)
    ├── Decimal (0..2)
    │   └── "42" (0..2)
    └── "nd" (2..4)`,
		},
		{
			Name: "Capture and backtrack",
			Grammar: `
                           G <- D '!' / D
                           D <- [0-9]+
                        `,
			Input:          "42",
			ExpectedCursor: 2,
			ExpectedAST: `G (0..2)
└── D (0..2)
    └── "42" (0..2)`,
		},
	}

	for _, test := range vmTests {
		t.Run("O0 "+test.Name, mkVmTestFn(test, 0))
		t.Run("O1 "+test.Name, mkVmTestFn(test, 1))
	}
}

func mkVmTestFn(test vmTest, opt int) func(t *testing.T) {
	return func(t *testing.T) {
		val, cur, err := exec(test.Grammar, test.Input, 0)

		// The cursor should be right for both error and
		// success states
		assert.Equal(t, test.ExpectedCursor, cur)

		// If the test contains an expected error, it will
		// take priority over asserting a success operation
		if test.ExpectedError != "" {
			require.Error(t, err)
			assert.Equal(t, test.ExpectedError, err.Error())
			return
		}

		// Finally testing the success state against the
		// output value, if there's any
		require.NoError(t, err)
		if test.ExpectedAST == "" {
			assert.Nil(t, val)
		} else {
			require.NotNil(t, val)
			assert.Equal(t, test.ExpectedAST, val.PrettyString())
		}
	}
}

func exec(expr, input string, optimize int) (Value, int, error) {
	ast, err := NewGrammarParser(expr).Parse()
	if err != nil {
		panic(err)
	}
	// ast, err = InjectWhitespaces(ast)
	// if err != nil {
	// 	panic(err)
	// }
	// ast, err = AddBuiltins(ast)
	// if err != nil {
	// 	panic(err)
	// }
	ast, err = AddCaptures(ast)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ast\n%s\n", ast.HighlightPrettyString())
	asm, err := Compile(ast, CompilerConfig{Optimize: optimize})
	if err != nil {
		panic(err)
	}
	fmt.Printf("asm\n%s\n", asm.HighlightPrettyString())

	code := Encode(asm)

	fmt.Printf("code\n%#v\n", code.code)

	return code.Match(strings.NewReader(input))
}
