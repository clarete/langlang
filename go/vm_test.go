package langlang

import (
	// "fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vmTest struct {
	Name           string
	Grammar        string
	Input          string
	ErrLabels      map[string]string
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

		vm := newVirtualMachine(bytecode, nil, nil, true)

		input := NewMemInput("")

		_, cur, err := vm.Match(&input)

		require.NoError(t, err)
		assert.Equal(t, 0, cur)
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
			ExpectedError:  "Unexpected '1' @ 1",
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
			ExpectedError:  "Unexpected ' ' @ 2",
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
		{
			Name: "Throw Cursor",
			Grammar: `
                           EXP <- OR (!PRI)^MissingOperator
                           OR  <- AND ("OR" AND)*
                           AND <- PRI ("AND" PRI)*
                           PRI <- '(' WRD ')'
                                / '!' WRD
                                / WRD
                           WRD <- "abacate"
                               / "abobora"
                               / "abadia"
                               / "abalado"
			`,
			Input:          `abacate abadia`,
			ExpectedCursor: 8,
			ExpectedError:  "Missing Operand Between Operators @ 9",
			ErrLabels: map[string]string{
				"MissingOperator": "Missing Operand Between Operators",
			},
		},
	}

	for _, test := range vmTests {
		t.Run("With_Charset_O0 "+test.Name, mkVmTestFn(test, 0, true))
		t.Run("With_Charset_O1 "+test.Name, mkVmTestFn(test, 1, true))
		t.Run("NO_Charset_O0 "+test.Name, mkVmTestFn(test, 0, false))
		t.Run("NO_Charset_O1 "+test.Name, mkVmTestFn(test, 1, false))
	}
}

func mkVmTestFn(test vmTest, optimize int, enableCharsets bool) func(t *testing.T) {
	return func(t *testing.T) {
		cfg := NewConfig()
		cfg.SetInt("compiler.optimize", optimize)
		cfg.SetBool("grammar.add_charsets", enableCharsets)

		ast, err := GrammarFromString(test.Grammar, cfg)
		if err != nil {
			panic(err)
		}
		// fmt.Printf("ast\n%s\n", ast.HighlightPrettyString())

		asm, err := Compile(ast, cfg)
		if err != nil {
			panic(err)
		}
		// fmt.Printf("asm\n%s\n", asm.HighlightPrettyString())

		code := Encode(asm)
		// fmt.Printf("code\n%#v\n", code.code)

		memInput := NewMemInput(test.Input)

		// Notice `showFails` is marked as false because `ClassNode`
		// and `CharsetNode` will generate different ordered choices.
		// The `ClassNode` variant generates a slice in the order it
		// was declared in the grammar.  The `CharsetNode` will use
		// the ascii table ordering.
		val, cur, err := code.MatchE(&memInput, test.ErrLabels, nil, false)

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
