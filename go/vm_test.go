package langlang

import (
	// "fmt"
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

		vm := newVirtualMachine(bytecode, map[string]string{}, map[string]struct{}{})

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
			ExpectedAST: `G (1..2)
└── "f" (1..2)`,
		},
		{
			Name:           "Any Star",
			Grammar:        "G <- .*",
			Input:          "foo",
			ExpectedCursor: 3,
			ExpectedAST: `G (1..4)
└── "foo" (1..4)`,
		},
		{
			Name:           "Char",
			Grammar:        "G <- 'f'",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST: `G (1..2)
└── "f" (1..2)`,
		},
		{
			Name:           "Choice",
			Grammar:        "G <- 'f' / 'g' / 'h'",
			Input:          "g",
			ExpectedCursor: 1,
			ExpectedAST: `G (1..2)
└── "g" (1..2)`,
		},
		{
			Name:           "Choice on Words",
			Grammar:        "G <- 'avocado' / 'avante' / 'aviador'",
			Input:          "avante",
			ExpectedCursor: 6,
			ExpectedAST: `G (1..7)
└── "avante" (1..7)`,
		},
		{
			Name:           "Class with Range",
			Grammar:        "G <- [0-9]+",
			Input:          "42",
			ExpectedCursor: 2,
			ExpectedAST: `G (1..3)
└── "42" (1..3)`,
		},
		{
			Name:           "Class with Range and Literal",
			Grammar:        "G <- [a-z_]+",
			Input:          "my_id",
			ExpectedCursor: 5,
			ExpectedAST: `G (1..6)
└── "my_id" (1..6)`,
		},
		{
			Name:           "Optional Matches",
			Grammar:        "G <- 'f'?",
			Input:          "foo",
			ExpectedCursor: 1,
			ExpectedAST: `G (1..2)
└── "f" (1..2)`,
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
			ExpectedAST: `G (1..4)
└── "bar" (1..4)`,
		},
		{
			Name:           "Not predicate",
			Grammar:        "G <- (!';' .)*",
			Input:          "foo; bar",
			ExpectedCursor: 3,
			ExpectedAST: `G (1..4)
└── "foo" (1..4)`,
		},
		{
			Name:           "Not Any and Star",
			Grammar:        `G <- "'" (!"'" .)* "'"`,
			Input:          "'foo'",
			ExpectedCursor: 5,
			ExpectedAST: `G (1..6)
└── "'foo'" (1..6)`,
		},
		{
			Name:           "And predicate",
			Grammar:        "G <- &'a' .",
			Input:          "avocado",
			ExpectedCursor: 1,
			ExpectedAST: `G (1..2)
└── "a" (1..2)`,
		},
		{
			Name:           "Parse HEX Number",
			Grammar:        "G <- '0x' [0-9a-fA-F]+ / '0'",
			Input:          "0xff",
			ExpectedCursor: 4,
			ExpectedAST: `G (1..5)
└── "0xff" (1..5)`,
		},
		{
			Name:           "Unicode",
			Grammar:        "G <- [♡]",
			Input:          "♡",
			ExpectedCursor: 3,
			ExpectedAST: `G (1..2)
└── "♡" (1..2)`,
		},
		{
			Name: "Var",
			Grammar: `G <- D
		D <- [0-9]+`,
			Input:          "1",
			ExpectedCursor: 1,
			ExpectedAST: `G (1..2)
└── D (1..2)
    └── "1" (1..2)`,
		},
		{
			Name: "Var and Var",
			Grammar: `G <- D P
				  D <- [0-9]+
				  P <- '!'`,
			Input:          "42!",
			ExpectedCursor: 3,
			ExpectedAST: `G (1..4)
└── Sequence<2> (1..4)
    ├── D (1..3)
    │   └── "42" (1..3)
    └── P (3..4)
        └── "!" (3..4)`,
		},
		{
			Name: "Var Char Var",
			Grammar: `G <- D '+' D
				  D <- [0-9]+`,
			Input:          "40+2",
			ExpectedCursor: 4,
			ExpectedAST: `G (1..5)
└── Sequence<3> (1..5)
    ├── D (1..3)
    │   └── "40" (1..3)
    ├── "+" (3..4)
    └── D (4..5)
        └── "2" (4..5)`,
		},
		{
			Name: "Var Char Var Char",
			Grammar: `G <- D '+' D '!'
				  D <- [0-9]+`,
			Input:          "40+2!",
			ExpectedCursor: 5,
			ExpectedAST: `G (1..6)
└── Sequence<4> (1..6)
    ├── D (1..3)
    │   └── "40" (1..3)
    ├── "+" (3..4)
    ├── D (4..5)
    │   └── "2" (4..5)
    └── "!" (5..6)`,
		},
		{
			Name: "Char Var Char",
			Grammar: `G <- '+' D '+'
				  D <- [0-9]+`,
			Input:          "+42+",
			ExpectedCursor: 4,
			ExpectedAST: `G (1..5)
└── Sequence<3> (1..5)
    ├── "+" (1..2)
    ├── D (2..4)
    │   └── "42" (2..4)
    └── "+" (4..5)`,
		},
		{
			Name: "Lexification", // TODO: Needs test for failure
			Grammar: `
                           Ordinal <- Decimal #('st' / 'nd' / 'rd' / 'th')
                           Decimal <- ([1-9][0-9]*) / '0'
                        `,
			Input:          "42nd",
			ExpectedCursor: 4,
			ExpectedAST: `Ordinal (1..5)
└── Sequence<2> (1..5)
    ├── Decimal (1..3)
    │   └── "42" (1..3)
    └── "nd" (3..5)`,
		},
		{
			Name: "Capture and backtrack",
			Grammar: `
                           G <- D '!' / D
                           D <- [0-9]+
                        `,
			Input:          "42",
			ExpectedCursor: 2,
			ExpectedAST: `G (1..3)
└── D (1..3)
    └── "42" (1..3)`,
		},
		{
			Name:           "Error on Char",
			Grammar:        `G <- 'a'`,
			Input:          "1",
			ExpectedCursor: 0,
			ExpectedError:  "Expected 'a' but got '1' @ 1",
		},
		{
			Name: "Lexification in sequence within sequence",
			Grammar: `
SPC0   <- #(Letter Alnum+) ":" #(Digit+)
SPC1   <- #Letter Alnum+ #":" Digit+
Alnum  <- [a-zA-Z0-9]
Letter <- [a-zA-Z]
Digit  <- [0-9]
`,
			Input:          "a 999:99",
			ExpectedCursor: 1,
			ExpectedError:  "Expected 'a-z', 'A-Z', '0-9' but got ' ' @ 2",
		},
		{
			Name: "Regression with JSON.String",
			Grammar: `
			   String <- '"' #(Char* '"')
			   Char   <- (!'"' .)
			`,
			Input:          `"f"`,
			ExpectedCursor: 3,
			ExpectedAST: `String (1..4)
└── Sequence<3> (1..4)
    ├── "\"" (1..2)
    ├── Char (2..3)
    │   └── "f" (2..3)
    └── "\"" (3..4)`,
		},
	}

	for _, test := range vmTests {
		t.Run("O0 "+test.Name, mkVmTestFn(test, 0))
		t.Run("O1 "+test.Name, mkVmTestFn(test, 1))
	}
}

func mkVmTestFn(test vmTest, opt int) func(t *testing.T) {
	return func(t *testing.T) {
		val, cur, err := exec(test.Grammar, test.Input, opt)

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
	ast, err = InjectWhitespaces(ast)
	if err != nil {
		panic(err)
	}
	ast, err = AddBuiltins(ast)
	if err != nil {
		panic(err)
	}
	ast, err = AddCaptures(ast)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("ast\n%s\n", ast.HighlightPrettyString())
	asm, err := Compile(ast, CompilerConfig{Optimize: optimize})
	if err != nil {
		panic(err)
	}
	// fmt.Printf("asm\n%s\n", asm.HighlightPrettyString())

	code := Encode(asm)

	// fmt.Printf("code\n%#v\n", code.code)

	return code.Match(strings.NewReader(input))
}
