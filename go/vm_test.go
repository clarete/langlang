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
		}}, NewConfig())
		assert.Equal(t, uint8(0), bytecode.code[0])

		vm := NewVirtualMachine(bytecode)
		vm.SetShowFails(true)

		input := []byte("")

		_, cur, err := vm.Match(input)

		require.NoError(t, err)
		assert.Equal(t, 0, cur)
	})

	t.Run("Char32 error message should not truncate expected rune", func(t *testing.T) {
		cfg := NewConfig()
		cfg.SetInt("compiler.optimize", 0)
		cfg.SetBool("grammar.add_charsets", true)

		loader := NewInMemoryImportLoader()
		loader.Add("test.peg", []byte("G <- '🧠'"))
		db := NewDatabase(cfg, loader)

		code, err := QueryBytecode(db, "test.peg")
		require.NoError(t, err)

		vm := NewVirtualMachine(code)
		vm.SetShowFails(true)

		_, cur, err := vm.Match([]byte("a"))
		require.Error(t, err)
		assert.Equal(t, 0, cur)
		assert.Equal(t, "Expected '🧠' but got 'a' @ 1", err.Error())
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
			Name:           "Unicode Char32 (emoji literal)",
			Grammar:        "G <- '🧠' 'a'",
			Input:          "🧠a",
			ExpectedCursor: 5,
			ExpectedAST: `G (1..3)
└── "🧠a" (1..3)`,
		},
		{
			Name:           "Unicode Range32 (emoji range)",
			Grammar:        "G <- [🧠-🧬]",
			Input:          "🧪",
			ExpectedCursor: 4,
			ExpectedAST: `G (1..2)
└── "🧪" (1..2)`,
		},
		// Japanese Unicode tests
		{
			Name:           "Unicode Japanese Hiragana literal",
			Grammar:        "G <- 'こんにちは'",
			Input:          "こんにちは",
			ExpectedCursor: 15, // 5 chars × 3 bytes each
			ExpectedAST: `G (1..6)
└── "こんにちは" (1..6)`,
		},
		{
			Name:           "Unicode Hiragana range",
			Grammar:        "G <- [ぁ-ん]+",
			Input:          "あいうえお",
			ExpectedCursor: 15, // 5 chars × 3 bytes each
			ExpectedAST: `G (1..6)
└── "あいうえお" (1..6)`,
		},
		{
			Name:           "Unicode Katakana range",
			Grammar:        "G <- [ァ-ン]+",
			Input:          "アイウエオ",
			ExpectedCursor: 15, // 5 chars × 3 bytes each
			ExpectedAST: `G (1..6)
└── "アイウエオ" (1..6)`,
		},
		{
			Name:           "Unicode Kanji range",
			Grammar:        "G <- [一-龯]+",
			Input:          "日本語",
			ExpectedCursor: 9, // 3 chars × 3 bytes each
			ExpectedAST: `G (1..4)
└── "日本語" (1..4)`,
		},
		// Korean Unicode tests
		{
			Name:           "Unicode Korean Hangul literal",
			Grammar:        "G <- '안녕하세요'",
			Input:          "안녕하세요",
			ExpectedCursor: 15, // 5 chars × 3 bytes each
			ExpectedAST: `G (1..6)
└── "안녕하세요" (1..6)`,
		},
		{
			Name:           "Unicode Hangul range",
			Grammar:        "G <- [가-힣]+",
			Input:          "한글",
			ExpectedCursor: 6, // 2 chars × 3 bytes each
			ExpectedAST: `G (1..3)
└── "한글" (1..3)`,
		},
		// Arabic Unicode tests
		{
			Name:           "Unicode Arabic literal",
			Grammar:        "G <- 'مرحبا'",
			Input:          "مرحبا",
			ExpectedCursor: 10, // 5 chars × 2 bytes each
			ExpectedAST: `G (1..6)
└── "مرحبا" (1..6)`,
		},
		{
			Name:           "Unicode Arabic range",
			Grammar:        "G <- [ء-ي]+",
			Input:          "عربي",
			ExpectedCursor: 8, // 4 chars × 2 bytes each
			ExpectedAST: `G (1..5)
└── "عربي" (1..5)`,
		},
		// Cyrillic/Russian Unicode tests
		{
			Name:           "Unicode Russian literal",
			Grammar:        "G <- 'привет'",
			Input:          "привет",
			ExpectedCursor: 12, // 6 chars × 2 bytes each
			ExpectedAST: `G (1..7)
└── "привет" (1..7)`,
		},
		{
			Name:           "Unicode Cyrillic range",
			Grammar:        "G <- [а-я]+",
			Input:          "мир",
			ExpectedCursor: 6, // 3 chars × 2 bytes each
			ExpectedAST: `G (1..4)
└── "мир" (1..4)`,
		},
		// Greek Unicode tests
		{
			Name:           "Unicode Greek range",
			Grammar:        "G <- [α-ω]+",
			Input:          "αβγ",
			ExpectedCursor: 6, // 3 chars × 2 bytes each
			ExpectedAST: `G (1..4)
└── "αβγ" (1..4)`,
		},
		// Mixed Unicode tests
		{
			Name:           "Unicode mixed ASCII and Japanese",
			Grammar:        "G <- [a-zA-Zあ-ん]+",
			Input:          "helloこんにちは",
			ExpectedCursor: 20, // 5 ASCII + 5 Hiragana × 3 bytes
			ExpectedAST: `G (1..11)
└── "helloこんにちは" (1..11)`,
		},
		{
			Name:           "Unicode multilingual identifier",
			Grammar:        "G <- [a-zA-Z_α-ωа-я]+",
			Input:          "hello_αβγмир",
			ExpectedCursor: 18, // 6 ASCII + 3 Greek × 2 + 3 Cyrillic × 2
			ExpectedAST: `G (1..13)
└── "hello_αβγмир" (1..13)`,
		},
		// More emoji tests
		{
			Name:           "Unicode emoji sequence",
			Grammar:        "G <- ('👍' / '👎' / '❤' / '😂')+",
			Input:          "👍👎❤",
			ExpectedCursor: 11, // 4 + 4 + 3 bytes
			ExpectedAST: `G (1..4)
└── "👍👎❤" (1..4)`,
		},
		{
			Name:           "Unicode emoji with text",
			Grammar:        "G <- 'I' ' ' '❤' ' ' 'Go'",
			Input:          "I ❤ Go",
			ExpectedCursor: 8, // 1 + 1 + 3 + 1 + 2 bytes
			ExpectedAST: `G (1..7)
└── "I ❤ Go" (1..7)`,
		},
		// Unicode escape sequence test
		{
			Name:           "Unicode escape sequence",
			Grammar:        "G <- '\\u{2661}'", // ♡
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
		// Left Recursion tests
		{
			Name:           "Left Recursion Basic",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion Chain of 3",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "n+n+n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..4)
    │   └── Sequence<3> (1..4)
    │       ├── E (1..2)
    │       │   └── "n" (1..2)
    │       ├── "+" (2..3)
    │       └── "n" (3..4)
    ├── "+" (4..5)
    └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Single Element",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "n",
			ExpectedCursor: 1,
			ExpectedAST: `E (1..2)
└── "n" (1..2)`,
		},
		{
			Name:           "Left Recursion Explicit Precedence",
			Grammar:        "E <- E¹ '+' 'n' / 'n'",
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion - Two E Calls",
			Grammar:        "E <- E '+' E / 'n'",
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── E (3..4)
        └── "n" (3..4)`,
		},
		{
			Name: "Left Recursion With Separate Digit Rule",
			Grammar: `E <- E '+' D / D
D <- '0' / '1' / '2'`,
			Input:          "1+2",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── D (1..2)
    │       └── "1" (1..2)
    ├── "+" (2..3)
    └── D (3..4)
        └── "2" (3..4)`,
		},
		{
			Name:           "Left Recursion With Operator Precedence",
			Grammar:        opPrecedenceGrammar,
			Input:          "n+n*n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── E (3..6)
        └── Sequence<3> (3..6)
            ├── E (3..4)
            │   └── "n" (3..4)
            ├── "*" (4..5)
            └── E (5..6)
                └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Chain of 5",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "n+n+n+n+n",
			ExpectedCursor: 9,
			ExpectedAST: `E (1..10)
└── Sequence<3> (1..10)
    ├── E (1..8)
    │   └── Sequence<3> (1..8)
    │       ├── E (1..6)
    │       │   └── Sequence<3> (1..6)
    │       │       ├── E (1..4)
    │       │       │   └── Sequence<3> (1..4)
    │       │       │       ├── E (1..2)
    │       │       │       │   └── "n" (1..2)
    │       │       │       ├── "+" (2..3)
    │       │       │       └── "n" (3..4)
    │       │       ├── "+" (4..5)
    │       │       └── "n" (5..6)
    │       ├── "+" (6..7)
    │       └── "n" (7..8)
    ├── "+" (8..9)
    └── "n" (9..10)`,
		},
		{
			Name:           "Left Recursion Error Non-matching",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "xyz",
			ExpectedCursor: 0,
			ExpectedError:  "Unexpected 'x' @ 1",
		},
		{
			Name:           "Left Recursion Partial Match",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "n+n+x",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion Two Ops Multiplication",
			Grammar:        "E <- E '+' E / E '*' E / 'n'",
			Input:          "n*n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "*" (2..3)
    └── E (3..4)
        └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion Two Ops Mixed n+n*n",
			Grammar:        "E <- E '+' E / E '*' E / 'n'",
			Input:          "n+n*n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── E (3..6)
        └── Sequence<3> (3..6)
            ├── E (3..4)
            │   └── "n" (3..4)
            ├── "*" (4..5)
            └── E (5..6)
                └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Two Ops Mixed n*n+n",
			Grammar:        "E <- E '+' E / E '*' E / 'n'",
			Input:          "n*n+n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "*" (2..3)
    └── E (3..6)
        └── Sequence<3> (3..6)
            ├── E (3..4)
            │   └── "n" (3..4)
            ├── "+" (4..5)
            └── E (5..6)
                └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Two Ops Longer Chain",
			Grammar:        "E <- E '+' E / E '*' E / 'n'",
			Input:          "n+n*n+n*n",
			ExpectedCursor: 9,
			ExpectedAST: `E (1..10)
└── Sequence<3> (1..10)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── E (3..10)
        └── Sequence<3> (3..10)
            ├── E (3..4)
            │   └── "n" (3..4)
            ├── "*" (4..5)
            └── E (5..10)
                └── Sequence<3> (5..10)
                    ├── E (5..6)
                    │   └── "n" (5..6)
                    ├── "+" (6..7)
                    └── E (7..10)
                        └── Sequence<3> (7..10)
                            ├── E (7..8)
                            │   └── "n" (7..8)
                            ├── "*" (8..9)
                            └── E (9..10)
                                └── "n" (9..10)`,
		},
		{
			Name:           "Left Recursion Explicit Prec n+n*n",
			Grammar:        "E <- E¹ '+' E² / E² '*' E³ / 'n'",
			Input:          "n+n*n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── E (3..6)
        └── Sequence<3> (3..6)
            ├── E (3..4)
            │   └── "n" (3..4)
            ├── "*" (4..5)
            └── E (5..6)
                └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Explicit Precedence n*n+n",
			Grammar:        "E <- E¹ '+' E² / E² '*' E³ / 'n'",
			Input:          "n*n+n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..4)
    │   └── Sequence<3> (1..4)
    │       ├── E (1..2)
    │       │   └── "n" (1..2)
    │       ├── "*" (2..3)
    │       └── E (3..4)
    │           └── "n" (3..4)
    ├── "+" (4..5)
    └── E (5..6)
        └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Arith Single Digit",
			Grammar:        arithmeticGrammar,
			Input:          "5",
			ExpectedCursor: 1,
			ExpectedAST: `Expr (1..2)
└── Term (1..2)
    └── Factor (1..2)
        └── "5" (1..2)`,
		},
		{
			Name:           "Left Recursion Arith Addition",
			Grammar:        arithmeticGrammar,
			Input:          "1+2",
			ExpectedCursor: 3,
			ExpectedAST: `Expr (1..4)
└── Sequence<3> (1..4)
    ├── Expr (1..2)
    │   └── Term (1..2)
    │       └── Factor (1..2)
    │           └── "1" (1..2)
    ├── "+" (2..3)
    └── Term (3..4)
        └── Factor (3..4)
            └── "2" (3..4)`,
		},
		{
			Name:           "Left Recursion Arith Multiplication",
			Grammar:        arithmeticGrammar,
			Input:          "3*4",
			ExpectedCursor: 3,
			ExpectedAST: `Expr (1..4)
└── Term (1..4)
    └── Sequence<3> (1..4)
        ├── Term (1..2)
        │   └── Factor (1..2)
        │       └── "3" (1..2)
        ├── "*" (2..3)
        └── Factor (3..4)
            └── "4" (3..4)`,
		},
		{
			Name:           "Left Recursion Arith Mixed Precedence",
			Grammar:        arithmeticGrammar,
			Input:          "1+2*3",
			ExpectedCursor: 5,
			ExpectedAST: `Expr (1..6)
└── Sequence<3> (1..6)
    ├── Expr (1..2)
    │   └── Term (1..2)
    │       └── Factor (1..2)
    │           └── "1" (1..2)
    ├── "+" (2..3)
    └── Term (3..6)
        └── Sequence<3> (3..6)
            ├── Term (3..4)
            │   └── Factor (3..4)
            │       └── "2" (3..4)
            ├── "*" (4..5)
            └── Factor (5..6)
                └── "3" (5..6)`,
		},
		{
			Name:           "Left Recursion Arith Parentheses",
			Grammar:        arithmeticGrammar,
			Input:          "(1+2)*3",
			ExpectedCursor: 7,
			ExpectedAST: `Expr (1..8)
└── Term (1..8)
    └── Sequence<3> (1..8)
        ├── Term (1..6)
        │   └── Factor (1..6)
        │       └── Sequence<3> (1..6)
        │           ├── "(" (1..2)
        │           ├── Expr (2..5)
        │           │   └── Sequence<3> (2..5)
        │           │       ├── Expr (2..3)
        │           │       │   └── Term (2..3)
        │           │       │       └── Factor (2..3)
        │           │       │           └── "1" (2..3)
        │           │       ├── "+" (3..4)
        │           │       └── Term (4..5)
        │           │           └── Factor (4..5)
        │           │               └── "2" (4..5)
        │           └── ")" (5..6)
        ├── "*" (6..7)
        └── Factor (7..8)
            └── "3" (7..8)`,
		},
		{
			Name:           "Left Recursion Arith Complex",
			Grammar:        arithmeticGrammar,
			Input:          "1+2*3+4",
			ExpectedCursor: 7,
			ExpectedAST: `Expr (1..8)
└── Sequence<3> (1..8)
    ├── Expr (1..6)
    │   └── Sequence<3> (1..6)
    │       ├── Expr (1..2)
    │       │   └── Term (1..2)
    │       │       └── Factor (1..2)
    │       │           └── "1" (1..2)
    │       ├── "+" (2..3)
    │       └── Term (3..6)
    │           └── Sequence<3> (3..6)
    │               ├── Term (3..4)
    │               │   └── Factor (3..4)
    │               │       └── "2" (3..4)
    │               ├── "*" (4..5)
    │               └── Factor (5..6)
    │                   └── "3" (5..6)
    ├── "+" (6..7)
    └── Term (7..8)
        └── Factor (7..8)
            └── "4" (7..8)`,
		},
		// Edge cases from arxiv 1207.0443
		{
			Name:           "Left Recursion Multiple Base Cases",
			Grammar:        "E <- E '+' E / '(' E ')' / 'n' / 'm'",
			Input:          "n+m",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── E (3..4)
        └── "m" (3..4)`,
		},
		{
			Name:           "Left Recursion Parenthesized Base",
			Grammar:        "E <- E '+' E / '(' E ')' / 'n'",
			Input:          "(n)+n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..4)
    │   └── Sequence<3> (1..4)
    │       ├── "(" (1..2)
    │       ├── E (2..3)
    │       │   └── "n" (2..3)
    │       └── ")" (3..4)
    ├── "+" (4..5)
    └── E (5..6)
        └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion With And Predicate",
			Grammar:        "E <- E '+' &'n' 'n' / 'n'",
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion With Not Predicate",
			Grammar:        "E <- E '+' !'*' 'n' / 'n'",
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion Not Predicate Blocks",
			Grammar:        "E <- E '+' !'n' 'n' / 'n'",
			Input:          "n+n",
			ExpectedCursor: 1,
			ExpectedAST: `E (1..2)
└── "n" (1..2)`,
		},
		{
			Name:           "Left Recursion With Lexification",
			Grammar:        "E <- E #('+' 'n') / 'n'",
			Input:          "n+n+n",
			ExpectedCursor: 5,
			ExpectedAST: `E (1..6)
└── Sequence<3> (1..6)
    ├── E (1..4)
    │   └── Sequence<3> (1..4)
    │       ├── E (1..2)
    │       │   └── "n" (1..2)
    │       ├── "+" (2..3)
    │       └── "n" (3..4)
    ├── "+" (4..5)
    └── "n" (5..6)`,
		},
		{
			Name:           "Left Recursion Base Only Matches",
			Grammar:        "E <- E '+' 'n' / 'n'",
			Input:          "n",
			ExpectedCursor: 1,
			ExpectedAST: `E (1..2)
└── "n" (1..2)`,
		},
		{
			Name: "Left Recursion Unary Prefix With Binary",
			Grammar: `E <- E¹ '+' E²
   / E¹ '-' E²
   / '-' E³
   / 'n'`,
			Input:          "-n+n",
			ExpectedCursor: 4,
			ExpectedAST: `E (1..5)
└── Sequence<3> (1..5)
    ├── E (1..3)
    │   └── Sequence<2> (1..3)
    │       ├── "-" (1..2)
    │       └── E (2..3)
    │           └── "n" (2..3)
    ├── "+" (3..4)
    └── E (4..5)
        └── "n" (4..5)`,
		},
		{
			Name: "Left Recursion Double Unary Prefix",
			Grammar: `E <- E '+' E
   / '-' E
   / 'n'`,
			Input:          "--n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<2> (1..4)
    ├── "-" (1..2)
    └── E (2..4)
        └── Sequence<2> (2..4)
            ├── "-" (2..3)
            └── E (3..4)
                └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion Longest Match",
			Grammar:        "E <- E '+' 'n' / E '+' / 'n'",
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `E (1..4)
└── Sequence<3> (1..4)
    ├── E (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Left Recursion Repetition After LR",
			Grammar:        "E <- E '+' 'n'+ / 'n'+",
			Input:          "nn+nnn",
			ExpectedCursor: 6,
			ExpectedAST: `E (1..7)
└── Sequence<5> (1..7)
    ├── E (1..3)
    │   └── Sequence<2> (1..3)
    │       ├── "n" (1..2)
    │       └── "n" (2..3)
    ├── "+" (3..4)
    ├── "n" (4..5)
    ├── "n" (5..6)
    └── "n" (6..7)`,
		},
		{
			Name: "Indirect Left Recursion",
			Grammar: `A <- B
B <- A '+' 'n' / 'n'`,
			Input:          "n+n+n",
			ExpectedCursor: 5,
			ExpectedAST: `A (1..6)
└── B (1..6)
    └── Sequence<3> (1..6)
        ├── A (1..4)
        │   └── B (1..4)
        │       └── Sequence<3> (1..4)
        │           ├── A (1..2)
        │           │   └── B (1..2)
        │           │       └── "n" (1..2)
        │           ├── "+" (2..3)
        │           └── "n" (3..4)
        ├── "+" (4..5)
        └── "n" (5..6)`,
		},
		// B? is nullable, so A effectively starts with itself when B fails to match
		{
			Name: "Hidden Left Recursion via Optional",
			Grammar: `A <- B? A '+' 'n' / 'n'
B <- 'x'`,
			Input:          "n+n",
			ExpectedCursor: 3,
			ExpectedAST: `A (1..4)
└── Sequence<3> (1..4)
    ├── A (1..2)
    │   └── "n" (1..2)
    ├── "+" (2..3)
    └── "n" (3..4)`,
		},
		{
			Name:           "Complement charset with multi-byte UTF-8 rune (opSet)",
			Grammar:        "G <- ![x] .",
			Input:          "☺",
			ExpectedCursor: 3,
			ExpectedAST: `G (1..2)
└── "☺" (1..2)`,
		},
		{
			Name:           "Complement charset span with multi-byte UTF-8 runes (opSpan)",
			Grammar:        "G <- (![;] .)*",
			Input:          "a☺b",
			ExpectedCursor: 5,
			ExpectedAST: `G (1..4)
└── "a☺b" (1..4)`,
		},
		{
			Name:           "Positive charset does not over-consume at UTF-8 boundary",
			Grammar:        "G <- [a-z]+ .",
			Input:          "abc☺",
			ExpectedCursor: 6,
			ExpectedAST: `G (1..5)
└── "abc☺" (1..5)`,
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

		loader := NewInMemoryImportLoader()
		loader.Add("test.peg", []byte(test.Grammar))
		db := NewDatabase(cfg, loader)

		code, err := QueryBytecode(db, "test.peg")
		if err != nil {
			panic(err)
		}

		input := []byte(test.Input)

		// Convert string-based error labels to int-based
		errLabels := code.CompileErrorLabels(test.ErrLabels)

		// Notice `showFails` is marked as false because `ClassNode`
		// and `CharsetNode` will generate different ordered choices.
		// The `ClassNode` variant generates a slice in the order it
		// was declared in the grammar.  The `CharsetNode` will use
		// the ascii table ordering.
		vm := NewVirtualMachine(code)
		vm.SetLabelMessages(errLabels)
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

var arithmeticGrammar = `
Expr   <- Expr '+' Term / Term
Term   <- Term '*' Factor / Factor
Factor <- '(' Expr ')' / [0-9]
`

var opPrecedenceGrammar = `
E <- E¹ '+' E²
   / E¹ '-' E²
   / E² '*' E³
   / E² '/' E³
   / '-' E⁴
   / '(' E¹ ')'
   / 'n'
`

func TestIndentOperators(t *testing.T) {
	mkVM := func(t *testing.T, grammar string) (*virtualMachine, *Database) {
		t.Helper()
		cfg := NewConfig()
		cfg.SetInt("compiler.optimize", 0)
		cfg.SetBool("grammar.add_charsets", false)
		loader := NewInMemoryImportLoader()
		loader.Add("test.peg", []byte(grammar))
		db := NewDatabase(cfg, loader)
		code, err := QueryBytecode(db, "test.peg")
		require.NoError(t, err)
		vm := NewVirtualMachine(code)
		return vm, db
	}

	indentGrammar := `
Program   <- Statement+ !.
Statement <- SAMEDENT (Block / Simple) NL?
Block     <- "if" NL INDENT Statement+ DEDENT
Simple    <- [a-z]+

Spacing   <- #([ \t])*
NL        <- #('\n' BlankLine*)
BlankLine <- #([ \t]* '\n')
`

	t.Run("valid single-level indent", func(t *testing.T) {
		vm, _ := mkVM(t, indentGrammar)
		input := "if\n  hello\n"
		_, _, err := vm.Match([]byte(input))
		require.NoError(t, err)
	})

	t.Run("valid two-level indent", func(t *testing.T) {
		vm, _ := mkVM(t, indentGrammar)
		input := "if\n  if\n    deep\n"
		_, _, err := vm.Match([]byte(input))
		require.NoError(t, err)
	})

	t.Run("reject invalid indent (not deeper)", func(t *testing.T) {
		rejectGrammar := `
Program   <- SAMEDENT "if" NL INDENT [a-z]+ DEDENT !.

Spacing   <- #([ \t])*
NL        <- #('\n')
`
		vm, _ := mkVM(t, rejectGrammar)
		input := "if\nhello"
		_, _, err := vm.Match([]byte(input))
		require.Error(t, err)
	})

	t.Run("multi-level dedent", func(t *testing.T) {
		vm, _ := mkVM(t, indentGrammar)
		input := "if\n  if\n    deep\n  back\n"
		_, _, err := vm.Match([]byte(input))
		require.NoError(t, err)
	})

	t.Run("samedent must match exactly", func(t *testing.T) {
		vm, _ := mkVM(t, indentGrammar)
		input := "  hello\n"
		_, _, err := vm.Match([]byte(input))
		require.Error(t, err)
	})

	t.Run("backtracking restores indent stack", func(t *testing.T) {
		backtrackGrammar := `
Program   <- Statement+ !.
Statement <- SAMEDENT (IfBlock / Simple) NL?
IfBlock   <- "if" NL INDENT Statement+ DEDENT
Simple    <- [a-z]+

Spacing   <- #([ \t])*
NL        <- #('\n' BlankLine*)
BlankLine <- #([ \t]* '\n')
`
		vm, _ := mkVM(t, backtrackGrammar)
		input := "hello\n"
		_, _, err := vm.Match([]byte(input))
		require.NoError(t, err)
	})

	t.Run("sequence of same-indent statements", func(t *testing.T) {
		vm, _ := mkVM(t, indentGrammar)
		input := "hello\nworld\n"
		_, _, err := vm.Match([]byte(input))
		require.NoError(t, err)
	})
}

func TestIndentUndefinedReferences(t *testing.T) {
	cfg := NewConfig()
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte("G <- INDENT [a-z]+ DEDENT\nH <- SAMEDENT [a-z]+"))
	db := NewDatabase(cfg, loader)

	refs, err := Get(db, UndefinedReferencesQuery, "test.peg")
	require.NoError(t, err)
	assert.Empty(t, refs, "INDENT/DEDENT/SAMEDENT should not be flagged as undefined")
}
