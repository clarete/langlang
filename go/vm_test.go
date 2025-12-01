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

// convertErrLabels converts string-based error label mapping to integer-based mapping
// using the bytecode's string table, extending it with new messages as needed
func convertErrLabels(strLabels map[string]string, code *Bytecode) map[int]int {
	if strLabels == nil {
		return nil
	}

	intLabels := make(map[int]int)
	for label, message := range strLabels {
		// Find the label ID in the string table
		labelID := -1
		messageID := -1

		for i, s := range code.strs {
			if s == label {
				labelID = i
			}
			if s == message {
				messageID = i
			}
		}

		// If we found the label but not the message, add the message to the table
		if labelID >= 0 {
			if messageID < 0 {
				// Message not in string table, so add it
				messageID = len(code.strs)
				code.strs = append(code.strs, message)
			}
			intLabels[labelID] = messageID
		}
	}

	return intLabels
}

func TestVM(t *testing.T) {
	t.Run("I guess I will just die", func(t *testing.T) {
		bytecode := Encode(&Program{code: []Instruction{
			IHalt{},
		}})
		assert.Equal(t, uint8(0), bytecode.code[0])

		vm := NewVirtualMachine(bytecode, nil, nil, true)

		input := []byte("")

		_, cur, err := vm.Match(input)

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
			ExpectedAST: `G (1..4)
└── "♡" (1..4)`,
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
			ExpectedError:  "Missing Operand Between Operators @ 9..15",
			ErrLabels: map[string]string{
				"MissingOperator": "Missing Operand Between Operators",
			},
		},
		{
			Name: "Space Injection within Repetition",
			Grammar: `
                           Val <- ID / Seq
                           Seq <- '(' Val* ')'   // <-- we should inject a space within this *
                           ID  <- [a-zA-Z_][a-zA-Z0-9_]*
			`,
			Input:          `(a b c)`,
			ExpectedCursor: 7,
			ExpectedAST: `Val (1..8)
└── Seq (1..8)
    └── Sequence<7> (1..8)
        ├── "(" (1..2)
        ├── Val (2..3)
        │   └── ID (2..3)
        │       └── "a" (2..3)
        ├── Spacing (3..4)
        │   └── " " (3..4)
        ├── Val (4..5)
        │   └── ID (4..5)
        │       └── "b" (4..5)
        ├── Spacing (5..6)
        │   └── " " (5..6)
        ├── Val (6..7)
        │   └── ID (6..7)
        │       └── "c" (6..7)
        └── ")" (7..8)`,
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

		ast, err := GrammarFromBytes([]byte(test.Grammar), cfg)
		if err != nil {
			panic(err)
		}
		// fmt.Printf("ast\n%s\n", ast.Highlight())

		asm, err := Compile(ast, cfg)
		if err != nil {
			panic(err)
		}
		// if test.Name == "Var" && optimize == 0 && enableCharsets {
		// 	fmt.Printf("asm\n%s\n", asm.Highlight())
		// }

		code := Encode(asm)

		// if test.Name == "Var" && optimize == 0 && enableCharsets {
		// 	fmt.Printf("strings: %v\n", code.strs)
		// }
		// fmt.Printf("code\n%#v\n", code.code)

		input := []byte(test.Input)

		// Convert string-based error labels to int-based
		errLabels := convertErrLabels(test.ErrLabels, code)

		// Notice `showFails` is marked as false because `ClassNode`
		// and `CharsetNode` will generate different ordered choices.
		// The `ClassNode` variant generates a slice in the order it
		// was declared in the grammar.  The `CharsetNode` will use
		// the ascii table ordering.
		vm := NewVirtualMachine(code, errLabels, nil, false)
		tree, cur, err := vm.Match(input)

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

		root, hasRoot := tree.Root()
		if test.ExpectedAST == "" {
			if hasRoot {
				assert.Equal(t, "", tree.Pretty(root))
			}
		} else {
			require.NotNil(t, tree)
			require.True(t, hasRoot, "expected tree to have a root")
			pretty := tree.Pretty(root)
			assert.Equal(t, test.ExpectedAST, pretty)
		}
	}
}
