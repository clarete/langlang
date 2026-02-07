package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ./arithmetic.peg -output-language go -output-path ./arithmetic.go -disable-capture-spaces
//go:generate go run ../../cmd/langlang -grammar ./arithmetic.peg -output-language go -output-path ./arithmetic.nocap.go -disable-capture-spaces -disable-captures -go-parser NoCapParser -go-remove-lib

func TestArithmeticSuccess(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:  "Simple number expression",
			Input: "42;",
			Expected: `Program (1..4)
└── Stmt (1..4)
    └── ExprStmt (1..4)
        └── Sequence<2> (1..4)
            ├── Expr (1..3)
            │   └── Number (1..3)
            │       └── "42" (1..3)
            └── ";" (3..4)`,
		},
		{
			Name:  "Simple addition",
			Input: "1 + 2;",
			Expected: `Program (1..7)
└── Stmt (1..7)
    └── ExprStmt (1..7)
        └── Sequence<2> (1..7)
            ├── Expr (1..6)
            │   └── Sequence<3> (1..6)
            │       ├── Expr (1..2)
            │       │   └── Number (1..2)
            │       │       └── "1" (1..2)
            │       ├── "+" (3..4)
            │       └── Expr (5..6)
            │           └── Number (5..6)
            │               └── "2" (5..6)
            └── ";" (6..7)`,
		},
		{
			Name:  "Multiplication binds tighter than addition",
			Input: "1 + 2 * 3;",
			Expected: `Program (1..11)
└── Stmt (1..11)
    └── ExprStmt (1..11)
        └── Sequence<2> (1..11)
            ├── Expr (1..10)
            │   └── Sequence<3> (1..10)
            │       ├── Expr (1..2)
            │       │   └── Number (1..2)
            │       │       └── "1" (1..2)
            │       ├── "+" (3..4)
            │       └── Expr (5..10)
            │           └── Sequence<3> (5..10)
            │               ├── Expr (5..6)
            │               │   └── Number (5..6)
            │               │       └── "2" (5..6)
            │               ├── "*" (7..8)
            │               └── Expr (9..10)
            │                   └── Number (9..10)
            │                       └── "3" (9..10)
            └── ";" (10..11)`,
		},
		{
			Name:  "Multiplication before addition in source",
			Input: "2 * 3 + 4;",
			Expected: `Program (1..11)
└── Stmt (1..11)
    └── ExprStmt (1..11)
        └── Sequence<2> (1..11)
            ├── Expr (1..10)
            │   └── Sequence<3> (1..10)
            │       ├── Expr (1..6)
            │       │   └── Sequence<3> (1..6)
            │       │       ├── Expr (1..2)
            │       │       │   └── Number (1..2)
            │       │       │       └── "2" (1..2)
            │       │       ├── "*" (3..4)
            │       │       └── Expr (5..6)
            │       │           └── Number (5..6)
            │       │               └── "3" (5..6)
            │       ├── "+" (7..8)
            │       └── Expr (9..10)
            │           └── Number (9..10)
            │               └── "4" (9..10)
            └── ";" (10..11)`,
		},
		{
			Name:  "Left-associative addition chain",
			Input: "1 + 2 + 3;",
			Expected: `Program (1..11)
└── Stmt (1..11)
    └── ExprStmt (1..11)
        └── Sequence<2> (1..11)
            ├── Expr (1..10)
            │   └── Sequence<3> (1..10)
            │       ├── Expr (1..6)
            │       │   └── Sequence<3> (1..6)
            │       │       ├── Expr (1..2)
            │       │       │   └── Number (1..2)
            │       │       │       └── "1" (1..2)
            │       │       ├── "+" (3..4)
            │       │       └── Expr (5..6)
            │       │           └── Number (5..6)
            │       │               └── "2" (5..6)
            │       ├── "+" (7..8)
            │       └── Expr (9..10)
            │           └── Number (9..10)
            │               └── "3" (9..10)
            └── ";" (10..11)`,
		},
		{
			Name:  "Complex mixed precedence chain",
			Input: "1 + 2 * 3 + 4;",
			Expected: `Program (1..15)
└── Stmt (1..15)
    └── ExprStmt (1..15)
        └── Sequence<2> (1..15)
            ├── Expr (1..14)
            │   └── Sequence<3> (1..14)
            │       ├── Expr (1..10)
            │       │   └── Sequence<3> (1..10)
            │       │       ├── Expr (1..2)
            │       │       │   └── Number (1..2)
            │       │       │       └── "1" (1..2)
            │       │       ├── "+" (3..4)
            │       │       └── Expr (5..10)
            │       │           └── Sequence<3> (5..10)
            │       │               ├── Expr (5..6)
            │       │               │   └── Number (5..6)
            │       │               │       └── "2" (5..6)
            │       │               ├── "*" (7..8)
            │       │               └── Expr (9..10)
            │       │                   └── Number (9..10)
            │       │                       └── "3" (9..10)
            │       ├── "+" (11..12)
            │       └── Expr (13..14)
            │           └── Number (13..14)
            │               └── "4" (13..14)
            └── ";" (14..15)`,
		},
		{
			Name:  "Parentheses override precedence",
			Input: "(1 + 2) * 3;",
			Expected: `Program (1..13)
└── Stmt (1..13)
    └── ExprStmt (1..13)
        └── Sequence<2> (1..13)
            ├── Expr (1..12)
            │   └── Sequence<3> (1..12)
            │       ├── Expr (1..8)
            │       │   └── Sequence<3> (1..8)
            │       │       ├── "(" (1..2)
            │       │       ├── Expr (2..7)
            │       │       │   └── Sequence<3> (2..7)
            │       │       │       ├── Expr (2..3)
            │       │       │       │   └── Number (2..3)
            │       │       │       │       └── "1" (2..3)
            │       │       │       ├── "+" (4..5)
            │       │       │       └── Expr (6..7)
            │       │       │           └── Number (6..7)
            │       │       │               └── "2" (6..7)
            │       │       └── ")" (7..8)
            │       ├── "*" (9..10)
            │       └── Expr (11..12)
            │           └── Number (11..12)
            │               └── "3" (11..12)
            └── ";" (12..13)`,
		},
		{
			Name:  "Unary negation",
			Input: "-1;",
			Expected: `Program (1..4)
└── Stmt (1..4)
    └── ExprStmt (1..4)
        └── Sequence<2> (1..4)
            ├── Expr (1..3)
            │   └── Sequence<2> (1..3)
            │       ├── "-" (1..2)
            │       └── Expr (2..3)
            │           └── Number (2..3)
            │               └── "1" (2..3)
            └── ";" (3..4)`,
		},
		{
			Name:  "Unary negation with addition",
			Input: "-1 + 2;",
			Expected: `Program (1..8)
└── Stmt (1..8)
    └── ExprStmt (1..8)
        └── Sequence<2> (1..8)
            ├── Expr (1..7)
            │   └── Sequence<3> (1..7)
            │       ├── Expr (1..3)
            │       │   └── Sequence<2> (1..3)
            │       │       ├── "-" (1..2)
            │       │       └── Expr (2..3)
            │       │           └── Number (2..3)
            │       │               └── "1" (2..3)
            │       ├── "+" (4..5)
            │       └── Expr (6..7)
            │           └── Number (6..7)
            │               └── "2" (6..7)
            └── ";" (7..8)`,
		},
		{
			Name:  "Division and modulo",
			Input: "10 / 2 % 3;",
			Expected: `Program (1..12)
└── Stmt (1..12)
    └── ExprStmt (1..12)
        └── Sequence<2> (1..12)
            ├── Expr (1..11)
            │   └── Sequence<3> (1..11)
            │       ├── Expr (1..7)
            │       │   └── Sequence<3> (1..7)
            │       │       ├── Expr (1..3)
            │       │       │   └── Number (1..3)
            │       │       │       └── "10" (1..3)
            │       │       ├── "/" (4..5)
            │       │       └── Expr (6..7)
            │       │           └── Number (6..7)
            │       │               └── "2" (6..7)
            │       ├── "%" (8..9)
            │       └── Expr (10..11)
            │           └── Number (10..11)
            │               └── "3" (10..11)
            └── ";" (11..12)`,
		},
		{
			Name:  "Comparison operator",
			Input: "1 < 2;",
			Expected: `Program (1..7)
└── Stmt (1..7)
    └── ExprStmt (1..7)
        └── Sequence<2> (1..7)
            ├── Expr (1..6)
            │   └── Sequence<3> (1..6)
            │       ├── Expr (1..2)
            │       │   └── Number (1..2)
            │       │       └── "1" (1..2)
            │       ├── "<" (3..4)
            │       └── Expr (5..6)
            │           └── Number (5..6)
            │               └── "2" (5..6)
            └── ";" (6..7)`,
		},
		{
			Name:  "Comparison with arithmetic",
			Input: "1 + 2 < 3 * 4;",
			Expected: `Program (1..15)
└── Stmt (1..15)
    └── ExprStmt (1..15)
        └── Sequence<2> (1..15)
            ├── Expr (1..14)
            │   └── Sequence<3> (1..14)
            │       ├── Expr (1..6)
            │       │   └── Sequence<3> (1..6)
            │       │       ├── Expr (1..2)
            │       │       │   └── Number (1..2)
            │       │       │       └── "1" (1..2)
            │       │       ├── "+" (3..4)
            │       │       └── Expr (5..6)
            │       │           └── Number (5..6)
            │       │               └── "2" (5..6)
            │       ├── "<" (7..8)
            │       └── Expr (9..14)
            │           └── Sequence<3> (9..14)
            │               ├── Expr (9..10)
            │               │   └── Number (9..10)
            │               │       └── "3" (9..10)
            │               ├── "*" (11..12)
            │               └── Expr (13..14)
            │                   └── Number (13..14)
            │                       └── "4" (13..14)
            └── ";" (14..15)`,
		},
		{
			Name:  "Equality operator",
			Input: "x == 0;",
			Expected: `Program (1..8)
└── Stmt (1..8)
    └── ExprStmt (1..8)
        └── Sequence<2> (1..8)
            ├── Expr (1..7)
            │   └── Sequence<3> (1..7)
            │       ├── Expr (1..2)
            │       │   └── Identifier (1..2)
            │       │       └── "x" (1..2)
            │       ├── "==" (3..5)
            │       └── Expr (6..7)
            │           └── Number (6..7)
            │               └── "0" (6..7)
            └── ";" (7..8)`,
		},
		{
			Name:  "Not-equal operator",
			Input: "x != y;",
			Expected: `Program (1..8)
└── Stmt (1..8)
    └── ExprStmt (1..8)
        └── Sequence<2> (1..8)
            ├── Expr (1..7)
            │   └── Sequence<3> (1..7)
            │       ├── Expr (1..2)
            │       │   └── Identifier (1..2)
            │       │       └── "x" (1..2)
            │       ├── "!=" (3..5)
            │       └── Expr (6..7)
            │           └── Identifier (6..7)
            │               └── "y" (6..7)
            └── ";" (7..8)`,
		},
		{
			Name:  "Logical and",
			Input: "a > 0 && b > 0;",
			Expected: `Program (1..16)
└── Stmt (1..16)
    └── ExprStmt (1..16)
        └── Sequence<2> (1..16)
            ├── Expr (1..15)
            │   └── Sequence<3> (1..15)
            │       ├── Expr (1..6)
            │       │   └── Sequence<3> (1..6)
            │       │       ├── Expr (1..2)
            │       │       │   └── Identifier (1..2)
            │       │       │       └── "a" (1..2)
            │       │       ├── ">" (3..4)
            │       │       └── Expr (5..6)
            │       │           └── Number (5..6)
            │       │               └── "0" (5..6)
            │       ├── "&&" (7..9)
            │       └── Expr (10..15)
            │           └── Sequence<3> (10..15)
            │               ├── Expr (10..11)
            │               │   └── Identifier (10..11)
            │               │       └── "b" (10..11)
            │               ├── ">" (12..13)
            │               └── Expr (14..15)
            │                   └── Number (14..15)
            │                       └── "0" (14..15)
            └── ";" (15..16)`,
		},
		{
			Name:  "Logical or",
			Input: "x == 1 || y == 2;",
			Expected: `Program (1..18)
└── Stmt (1..18)
    └── ExprStmt (1..18)
        └── Sequence<2> (1..18)
            ├── Expr (1..17)
            │   └── Sequence<3> (1..17)
            │       ├── Expr (1..7)
            │       │   └── Sequence<3> (1..7)
            │       │       ├── Expr (1..2)
            │       │       │   └── Identifier (1..2)
            │       │       │       └── "x" (1..2)
            │       │       ├── "==" (3..5)
            │       │       └── Expr (6..7)
            │       │           └── Number (6..7)
            │       │               └── "1" (6..7)
            │       ├── "||" (8..10)
            │       └── Expr (11..17)
            │           └── Sequence<3> (11..17)
            │               ├── Expr (11..12)
            │               │   └── Identifier (11..12)
            │               │       └── "y" (11..12)
            │               ├── "==" (13..15)
            │               └── Expr (16..17)
            │                   └── Number (16..17)
            │                       └── "2" (16..17)
            └── ";" (17..18)`,
		},
		{
			Name:  "Full precedence chain",
			Input: "a + b * c == d || e && f < g;",
			Expected: `Program (1..30)
└── Stmt (1..30)
    └── ExprStmt (1..30)
        └── Sequence<2> (1..30)
            ├── Expr (1..29)
            │   └── Sequence<3> (1..29)
            │       ├── Expr (1..15)
            │       │   └── Sequence<3> (1..15)
            │       │       ├── Expr (1..10)
            │       │       │   └── Sequence<3> (1..10)
            │       │       │       ├── Expr (1..2)
            │       │       │       │   └── Identifier (1..2)
            │       │       │       │       └── "a" (1..2)
            │       │       │       ├── "+" (3..4)
            │       │       │       └── Expr (5..10)
            │       │       │           └── Sequence<3> (5..10)
            │       │       │               ├── Expr (5..6)
            │       │       │               │   └── Identifier (5..6)
            │       │       │               │       └── "b" (5..6)
            │       │       │               ├── "*" (7..8)
            │       │       │               └── Expr (9..10)
            │       │       │                   └── Identifier (9..10)
            │       │       │                       └── "c" (9..10)
            │       │       ├── "==" (11..13)
            │       │       └── Expr (14..15)
            │       │           └── Identifier (14..15)
            │       │               └── "d" (14..15)
            │       ├── "||" (16..18)
            │       └── Expr (19..29)
            │           └── Sequence<3> (19..29)
            │               ├── Expr (19..20)
            │               │   └── Identifier (19..20)
            │               │       └── "e" (19..20)
            │               ├── "&&" (21..23)
            │               └── Expr (24..29)
            │                   └── Sequence<3> (24..29)
            │                       ├── Expr (24..25)
            │                       │   └── Identifier (24..25)
            │                       │       └── "f" (24..25)
            │                       ├── "<" (26..27)
            │                       └── Expr (28..29)
            │                           └── Identifier (28..29)
            │                               └── "g" (28..29)
            └── ";" (29..30)`,
		},
		{
			Name:  "Less-than-or-equal",
			Input: "x <= 10;",
			Expected: `Program (1..9)
└── Stmt (1..9)
    └── ExprStmt (1..9)
        └── Sequence<2> (1..9)
            ├── Expr (1..8)
            │   └── Sequence<3> (1..8)
            │       ├── Expr (1..2)
            │       │   └── Identifier (1..2)
            │       │       └── "x" (1..2)
            │       ├── "<=" (3..5)
            │       └── Expr (6..8)
            │           └── Number (6..8)
            │               └── "10" (6..8)
            └── ";" (8..9)`,
		},
		{
			Name:  "Greater-than-or-equal",
			Input: "x >= 0;",
			Expected: `Program (1..8)
└── Stmt (1..8)
    └── ExprStmt (1..8)
        └── Sequence<2> (1..8)
            ├── Expr (1..7)
            │   └── Sequence<3> (1..7)
            │       ├── Expr (1..2)
            │       │   └── Identifier (1..2)
            │       │       └── "x" (1..2)
            │       ├── ">=" (3..5)
            │       └── Expr (6..7)
            │           └── Number (6..7)
            │               └── "0" (6..7)
            └── ";" (7..8)`,
		},
		{
			Name:  "Nested parentheses",
			Input: "((1 + 2));",
			Expected: `Program (1..11)
└── Stmt (1..11)
    └── ExprStmt (1..11)
        └── Sequence<2> (1..11)
            ├── Expr (1..10)
            │   └── Sequence<3> (1..10)
            │       ├── "(" (1..2)
            │       ├── Expr (2..9)
            │       │   └── Sequence<3> (2..9)
            │       │       ├── "(" (2..3)
            │       │       ├── Expr (3..8)
            │       │       │   └── Sequence<3> (3..8)
            │       │       │       ├── Expr (3..4)
            │       │       │       │   └── Number (3..4)
            │       │       │       │       └── "1" (3..4)
            │       │       │       ├── "+" (5..6)
            │       │       │       └── Expr (7..8)
            │       │       │           └── Number (7..8)
            │       │       │               └── "2" (7..8)
            │       │       └── ")" (8..9)
            │       └── ")" (9..10)
            └── ";" (10..11)`,
		},
		{
			Name:  "Let statement simple",
			Input: "let x = 42;",
			Expected: `Program (1..12)
└── Stmt (1..12)
    └── LetStmt (1..12)
        └── Sequence<5> (1..12)
            ├── "let" (1..4)
            ├── Identifier (5..6)
            │   └── "x" (5..6)
            ├── "=" (7..8)
            ├── Expr (9..11)
            │   └── Number (9..11)
            │       └── "42" (9..11)
            └── ";" (11..12)`,
		},
		{
			Name:  "Let statement with expression",
			Input: "let result = 1 + 2 * 3;",
			Expected: `Program (1..24)
└── Stmt (1..24)
    └── LetStmt (1..24)
        └── Sequence<5> (1..24)
            ├── "let" (1..4)
            ├── Identifier (5..11)
            │   └── "result" (5..11)
            ├── "=" (12..13)
            ├── Expr (14..23)
            │   └── Sequence<3> (14..23)
            │       ├── Expr (14..15)
            │       │   └── Number (14..15)
            │       │       └── "1" (14..15)
            │       ├── "+" (16..17)
            │       └── Expr (18..23)
            │           └── Sequence<3> (18..23)
            │               ├── Expr (18..19)
            │               │   └── Number (18..19)
            │               │       └── "2" (18..19)
            │               ├── "*" (20..21)
            │               └── Expr (22..23)
            │                   └── Number (22..23)
            │                       └── "3" (22..23)
            └── ";" (23..24)`,
		},
		{
			Name:  "If statement",
			Input: "if (x > 0) { x; }",
			Expected: `Program (1..18)
└── Stmt (1..18)
    └── IfStmt (1..18)
        └── Sequence<7> (1..18)
            ├── "if" (1..3)
            ├── "(" (4..5)
            ├── Expr (5..10)
            │   └── Sequence<3> (5..10)
            │       ├── Expr (5..6)
            │       │   └── Identifier (5..6)
            │       │       └── "x" (5..6)
            │       ├── ">" (7..8)
            │       └── Expr (9..10)
            │           └── Number (9..10)
            │               └── "0" (9..10)
            ├── ")" (10..11)
            ├── "{" (12..13)
            ├── Stmt (14..16)
            │   └── ExprStmt (14..16)
            │       └── Sequence<2> (14..16)
            │           ├── Expr (14..15)
            │           │   └── Identifier (14..15)
            │           │       └── "x" (14..15)
            │           └── ";" (15..16)
            └── "}" (17..18)`,
		},
		{
			Name:  "Multiple statements",
			Input: "let x = 1; let y = x + 2; y * 3;",
			Expected: `Program (1..33)
└── Sequence<3> (1..33)
    ├── Stmt (1..11)
    │   └── LetStmt (1..11)
    │       └── Sequence<5> (1..11)
    │           ├── "let" (1..4)
    │           ├── Identifier (5..6)
    │           │   └── "x" (5..6)
    │           ├── "=" (7..8)
    │           ├── Expr (9..10)
    │           │   └── Number (9..10)
    │           │       └── "1" (9..10)
    │           └── ";" (10..11)
    ├── Stmt (12..26)
    │   └── LetStmt (12..26)
    │       └── Sequence<5> (12..26)
    │           ├── "let" (12..15)
    │           ├── Identifier (16..17)
    │           │   └── "y" (16..17)
    │           ├── "=" (18..19)
    │           ├── Expr (20..25)
    │           │   └── Sequence<3> (20..25)
    │           │       ├── Expr (20..21)
    │           │       │   └── Identifier (20..21)
    │           │       │       └── "x" (20..21)
    │           │       ├── "+" (22..23)
    │           │       └── Expr (24..25)
    │           │           └── Number (24..25)
    │           │               └── "2" (24..25)
    │           └── ";" (25..26)
    └── Stmt (27..33)
        └── ExprStmt (27..33)
            └── Sequence<2> (27..33)
                ├── Expr (27..32)
                │   └── Sequence<3> (27..32)
                │       ├── Expr (27..28)
                │       │   └── Identifier (27..28)
                │       │       └── "y" (27..28)
                │       ├── "*" (29..30)
                │       └── Expr (31..32)
                │           └── Number (31..32)
                │               └── "3" (31..32)
                └── ";" (32..33)`,
		},
		{
			Name:  "If with multiple body statements",
			Input: "if (a > b) { let t = a; a; }",
			Expected: `Program (1..29)
└── Stmt (1..29)
    └── IfStmt (1..29)
        └── Sequence<8> (1..29)
            ├── "if" (1..3)
            ├── "(" (4..5)
            ├── Expr (5..10)
            │   └── Sequence<3> (5..10)
            │       ├── Expr (5..6)
            │       │   └── Identifier (5..6)
            │       │       └── "a" (5..6)
            │       ├── ">" (7..8)
            │       └── Expr (9..10)
            │           └── Identifier (9..10)
            │               └── "b" (9..10)
            ├── ")" (10..11)
            ├── "{" (12..13)
            ├── Stmt (14..24)
            │   └── LetStmt (14..24)
            │       └── Sequence<5> (14..24)
            │           ├── "let" (14..17)
            │           ├── Identifier (18..19)
            │           │   └── "t" (18..19)
            │           ├── "=" (20..21)
            │           ├── Expr (22..23)
            │           │   └── Identifier (22..23)
            │           │       └── "a" (22..23)
            │           └── ";" (23..24)
            ├── Stmt (25..27)
            │   └── ExprStmt (25..27)
            │       └── Sequence<2> (25..27)
            │           ├── Expr (25..26)
            │           │   └── Identifier (25..26)
            │           │       └── "a" (25..26)
            │           └── ";" (26..27)
            └── "}" (28..29)`,
		},
		{
			Name:  "Deeply nested arithmetic",
			Input: "1 + 2 * 3 - 4 / 2 + 5 % 3;",
			Expected: `Program (1..27)
└── Stmt (1..27)
    └── ExprStmt (1..27)
        └── Sequence<2> (1..27)
            ├── Expr (1..26)
            │   └── Sequence<3> (1..26)
            │       ├── Expr (1..18)
            │       │   └── Sequence<3> (1..18)
            │       │       ├── Expr (1..10)
            │       │       │   └── Sequence<3> (1..10)
            │       │       │       ├── Expr (1..2)
            │       │       │       │   └── Number (1..2)
            │       │       │       │       └── "1" (1..2)
            │       │       │       ├── "+" (3..4)
            │       │       │       └── Expr (5..10)
            │       │       │           └── Sequence<3> (5..10)
            │       │       │               ├── Expr (5..6)
            │       │       │               │   └── Number (5..6)
            │       │       │               │       └── "2" (5..6)
            │       │       │               ├── "*" (7..8)
            │       │       │               └── Expr (9..10)
            │       │       │                   └── Number (9..10)
            │       │       │                       └── "3" (9..10)
            │       │       ├── "-" (11..12)
            │       │       └── Expr (13..18)
            │       │           └── Sequence<3> (13..18)
            │       │               ├── Expr (13..14)
            │       │               │   └── Number (13..14)
            │       │               │       └── "4" (13..14)
            │       │               ├── "/" (15..16)
            │       │               └── Expr (17..18)
            │       │                   └── Number (17..18)
            │       │                       └── "2" (17..18)
            │       ├── "+" (19..20)
            │       └── Expr (21..26)
            │           └── Sequence<3> (21..26)
            │               ├── Expr (21..22)
            │               │   └── Number (21..22)
            │               │       └── "5" (21..22)
            │               ├── "%" (23..24)
            │               └── Expr (25..26)
            │                   └── Number (25..26)
            │                       └── "3" (25..26)
            └── ";" (26..27)`,
		},
		{
			Name:  "Comparison chains with logical operators",
			Input: "a > 0 && b < 10 || c == 5;",
			Expected: `Program (1..27)
└── Stmt (1..27)
    └── ExprStmt (1..27)
        └── Sequence<2> (1..27)
            ├── Expr (1..26)
            │   └── Sequence<3> (1..26)
            │       ├── Expr (1..16)
            │       │   └── Sequence<3> (1..16)
            │       │       ├── Expr (1..6)
            │       │       │   └── Sequence<3> (1..6)
            │       │       │       ├── Expr (1..2)
            │       │       │       │   └── Identifier (1..2)
            │       │       │       │       └── "a" (1..2)
            │       │       │       ├── ">" (3..4)
            │       │       │       └── Expr (5..6)
            │       │       │           └── Number (5..6)
            │       │       │               └── "0" (5..6)
            │       │       ├── "&&" (7..9)
            │       │       └── Expr (10..16)
            │       │           └── Sequence<3> (10..16)
            │       │               ├── Expr (10..11)
            │       │               │   └── Identifier (10..11)
            │       │               │       └── "b" (10..11)
            │       │               ├── "<" (12..13)
            │       │               └── Expr (14..16)
            │       │                   └── Number (14..16)
            │       │                       └── "10" (14..16)
            │       ├── "||" (17..19)
            │       └── Expr (20..26)
            │           └── Sequence<3> (20..26)
            │               ├── Expr (20..21)
            │               │   └── Identifier (20..21)
            │               │       └── "c" (20..21)
            │               ├── "==" (22..24)
            │               └── Expr (25..26)
            │                   └── Number (25..26)
            │                       └── "5" (25..26)
            └── ";" (26..27)`,
		},
		{
			Name:  "Identifier expression",
			Input: "foo;",
			Expected: `Program (1..5)
└── Stmt (1..5)
    └── ExprStmt (1..5)
        └── Sequence<2> (1..5)
            ├── Expr (1..4)
            │   └── Identifier (1..4)
            │       └── "foo" (1..4)
            └── ";" (4..5)`,
		},
		{
			Name:  "Zero literal",
			Input: "0;",
			Expected: `Program (1..3)
└── Stmt (1..3)
    └── ExprStmt (1..3)
        └── Sequence<2> (1..3)
            ├── Expr (1..2)
            │   └── Number (1..2)
            │       └── "0" (1..2)
            └── ";" (2..3)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Input))
			v, err := p.ParseProgram()
			require.NoError(t, err)
			root, hasRoot := v.Root()
			require.True(t, hasRoot)
			assert.Equal(t, test.Expected, v.Pretty(root))
		})
	}
}

func TestArithmeticRecovery(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:  "Missing semicolon in let",
			Input: "let x = 1",
			Expected: `Program (1..10)
└── Stmt (1..10)
    └── LetStmt (1..10)
        └── Sequence<5> (1..10)
            ├── "let" (1..4)
            ├── Identifier (5..6)
            │   └── "x" (5..6)
            ├── "=" (7..8)
            ├── Expr (9..10)
            │   └── Number (9..10)
            │       └── "1" (9..10)
            └── Error<letsemi> (10)`,
		},
		{
			Name:  "Missing semicolon in expression",
			Input: "42",
			Expected: `Program (1..3)
└── Stmt (1..3)
    └── ExprStmt (1..3)
        └── Sequence<2> (1..3)
            ├── Expr (1..3)
            │   └── Number (1..3)
            │       └── "42" (1..3)
            └── Error<exprsemi> (3)`,
		},
		{
			Name:  "Missing close paren",
			Input: "(1 + 2;",
			Expected: `Program (1..8)
└── Stmt (1..8)
    └── ExprStmt (1..8)
        └── Sequence<2> (1..8)
            ├── Expr (1..7)
            │   └── Sequence<3> (1..7)
            │       ├── "(" (1..2)
            │       ├── Expr (2..7)
            │       │   └── Sequence<3> (2..7)
            │       │       ├── Expr (2..3)
            │       │       │   └── Number (2..3)
            │       │       │       └── "1" (2..3)
            │       │       ├── "+" (4..5)
            │       │       └── Expr (6..7)
            │       │           └── Number (6..7)
            │       │               └── "2" (6..7)
            │       └── Error<closeparen> (7)
            └── ";" (7..8)`,
		},
		{
			Name:  "Missing right curly in if",
			Input: "if (x > 0) { x;",
			Expected: `Program (1..16)
└── Stmt (1..16)
    └── IfStmt (1..16)
        └── Sequence<7> (1..16)
            ├── "if" (1..3)
            ├── "(" (4..5)
            ├── Expr (5..10)
            │   └── Sequence<3> (5..10)
            │       ├── Expr (5..6)
            │       │   └── Identifier (5..6)
            │       │       └── "x" (5..6)
            │       ├── ">" (7..8)
            │       └── Expr (9..10)
            │           └── Number (9..10)
            │               └── "0" (9..10)
            ├── ")" (10..11)
            ├── "{" (12..13)
            ├── Stmt (14..16)
            │   └── ExprStmt (14..16)
            │       └── Sequence<2> (14..16)
            │           ├── Expr (14..15)
            │           │   └── Identifier (14..15)
            │           │       └── "x" (14..15)
            │           └── ";" (15..16)
            └── Error<ifrcurl> (16)`,
		},
		{
			Name:  "Missing if left paren",
			Input: "if x > 0) { x; }",
			Expected: `Program (1..17)
└── Stmt (1..17)
    └── IfStmt (1..17)
        └── Sequence<7> (1..17)
            ├── "if" (1..3)
            ├── Error<iflpar> (4)
            ├── Expr (4..9)
            │   └── Sequence<3> (4..9)
            │       ├── Expr (4..5)
            │       │   └── Identifier (4..5)
            │       │       └── "x" (4..5)
            │       ├── ">" (6..7)
            │       └── Expr (8..9)
            │           └── Number (8..9)
            │               └── "0" (8..9)
            ├── ")" (9..10)
            ├── "{" (11..12)
            ├── Stmt (13..15)
            │   └── ExprStmt (13..15)
            │       └── Sequence<2> (13..15)
            │           ├── Expr (13..14)
            │           │   └── Identifier (13..14)
            │           │       └── "x" (13..14)
            │           └── ";" (14..15)
            └── "}" (16..17)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Input))
			v, err := p.ParseProgram()
			require.NoError(t, err)
			require.NotNil(t, v)
			root, hasRoot := v.Root()
			require.True(t, hasRoot)
			assert.Equal(t, test.Expected, v.Pretty(root))
		})
	}
}

// Benchmark scenarios of increasing complexity to stress left
// recursion with operator precedence, recovery, and spacing.
type benchScenario struct {
	Name  string
	Input []byte
}

var benchmarks = []benchScenario{
	{
		Name:  "Atom",
		Input: []byte("42;"),
	},
	{
		Name:  "Addition",
		Input: []byte("1 + 2;"),
	},
	{
		Name:  "Mixed Precedence",
		Input: []byte("1 + 2 * 3;"),
	},
	{
		Name:  "Parenthesized",
		Input: []byte("(1 + 2) * 3;"),
	},
	{
		Name:  "Unary Plus Binary",
		Input: []byte("-1 + 2;"),
	},
	{
		Name:  "Deep Arithmetic",
		Input: []byte("1 + 2 * 3 - 4 / 2 + 5 % 3;"),
	},
	{
		Name:  "Logical And Comparison",
		Input: []byte("a > 0 && b < 10 || c == 5;"),
	},
	{
		Name:  "Full Precedence Chain",
		Input: []byte("a + b * c == d || e && f < g;"),
	},
	{
		Name:  "Let Statement",
		Input: []byte("let result = 1 + 2 * 3;"),
	},
	{
		Name:  "If Statement",
		Input: []byte("if (x > 0) { x; }"),
	},
	{
		Name:  "Multi Statement",
		Input: []byte("let x = 1; let y = x + 2; y * 3;"),
	},
	{
		Name:  "Complex Program",
		Input: []byte("let a = 1 + 2 * 3; let b = (a + 4) / 2; if (a > b && b != 0) { a + b * 2; }"),
	},
}

func BenchmarkParser(b *testing.B) {
	p := NewParser()
	p.SetShowFails(false)

	for _, scenario := range benchmarks {
		b.Run(scenario.Name, func(b *testing.B) {
			b.SetBytes(int64(len(scenario.Input)))
			p.SetInput(scenario.Input)

			for n := 0; n < b.N; n++ {
				p.ParseProgram()
			}
		})
	}
}

func BenchmarkNoCapParser(b *testing.B) {
	p := NewNoCapParser()
	p.SetShowFails(false)

	for _, scenario := range benchmarks {
		b.Run(scenario.Name, func(b *testing.B) {
			b.SetBytes(int64(len(scenario.Input)))
			p.SetInput(scenario.Input)

			for n := 0; n < b.N; n++ {
				p.ParseProgram()
			}
		})
	}
}
