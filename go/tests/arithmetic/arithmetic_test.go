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
            │   └── LogicalOrExpr (1..3)
            │       └── LogicalAndExpr (1..3)
            │           └── EqualityExpr (1..3)
            │               └── ComparisonExpr (1..3)
            │                   └── AdditiveExpr (1..3)
            │                       └── MultiplicativeExpr (1..3)
            │                           └── UnaryExpr (1..3)
            │                               └── PrimaryExpr (1..3)
            │                                   └── Number (1..3)
            │                                       └── "42" (1..3)
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
            │   └── LogicalOrExpr (1..6)
            │       └── LogicalAndExpr (1..6)
            │           └── EqualityExpr (1..6)
            │               └── ComparisonExpr (1..6)
            │                   └── AdditiveExpr (1..6)
            │                       └── Sequence<3> (1..6)
            │                           ├── MultiplicativeExpr (1..2)
            │                           │   └── UnaryExpr (1..2)
            │                           │       └── PrimaryExpr (1..2)
            │                           │           └── Number (1..2)
            │                           │               └── "1" (1..2)
            │                           ├── "+" (3..4)
            │                           └── MultiplicativeExpr (5..6)
            │                               └── UnaryExpr (5..6)
            │                                   └── PrimaryExpr (5..6)
            │                                       └── Number (5..6)
            │                                           └── "2" (5..6)
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
            │   └── LogicalOrExpr (1..10)
            │       └── LogicalAndExpr (1..10)
            │           └── EqualityExpr (1..10)
            │               └── ComparisonExpr (1..10)
            │                   └── AdditiveExpr (1..10)
            │                       └── Sequence<3> (1..10)
            │                           ├── MultiplicativeExpr (1..2)
            │                           │   └── UnaryExpr (1..2)
            │                           │       └── PrimaryExpr (1..2)
            │                           │           └── Number (1..2)
            │                           │               └── "1" (1..2)
            │                           ├── "+" (3..4)
            │                           └── MultiplicativeExpr (5..10)
            │                               └── Sequence<3> (5..10)
            │                                   ├── UnaryExpr (5..6)
            │                                   │   └── PrimaryExpr (5..6)
            │                                   │       └── Number (5..6)
            │                                   │           └── "2" (5..6)
            │                                   ├── "*" (7..8)
            │                                   └── UnaryExpr (9..10)
            │                                       └── PrimaryExpr (9..10)
            │                                           └── Number (9..10)
            │                                               └── "3" (9..10)
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
            │   └── LogicalOrExpr (1..10)
            │       └── LogicalAndExpr (1..10)
            │           └── EqualityExpr (1..10)
            │               └── ComparisonExpr (1..10)
            │                   └── AdditiveExpr (1..10)
            │                       └── Sequence<3> (1..10)
            │                           ├── MultiplicativeExpr (1..6)
            │                           │   └── Sequence<3> (1..6)
            │                           │       ├── UnaryExpr (1..2)
            │                           │       │   └── PrimaryExpr (1..2)
            │                           │       │       └── Number (1..2)
            │                           │       │           └── "2" (1..2)
            │                           │       ├── "*" (3..4)
            │                           │       └── UnaryExpr (5..6)
            │                           │           └── PrimaryExpr (5..6)
            │                           │               └── Number (5..6)
            │                           │                   └── "3" (5..6)
            │                           ├── "+" (7..8)
            │                           └── MultiplicativeExpr (9..10)
            │                               └── UnaryExpr (9..10)
            │                                   └── PrimaryExpr (9..10)
            │                                       └── Number (9..10)
            │                                           └── "4" (9..10)
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
            │   └── LogicalOrExpr (1..10)
            │       └── LogicalAndExpr (1..10)
            │           └── EqualityExpr (1..10)
            │               └── ComparisonExpr (1..10)
            │                   └── AdditiveExpr (1..10)
            │                       └── Sequence<5> (1..10)
            │                           ├── MultiplicativeExpr (1..2)
            │                           │   └── UnaryExpr (1..2)
            │                           │       └── PrimaryExpr (1..2)
            │                           │           └── Number (1..2)
            │                           │               └── "1" (1..2)
            │                           ├── "+" (3..4)
            │                           ├── MultiplicativeExpr (5..6)
            │                           │   └── UnaryExpr (5..6)
            │                           │       └── PrimaryExpr (5..6)
            │                           │           └── Number (5..6)
            │                           │               └── "2" (5..6)
            │                           ├── "+" (7..8)
            │                           └── MultiplicativeExpr (9..10)
            │                               └── UnaryExpr (9..10)
            │                                   └── PrimaryExpr (9..10)
            │                                       └── Number (9..10)
            │                                           └── "3" (9..10)
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
            │   └── LogicalOrExpr (1..14)
            │       └── LogicalAndExpr (1..14)
            │           └── EqualityExpr (1..14)
            │               └── ComparisonExpr (1..14)
            │                   └── AdditiveExpr (1..14)
            │                       └── Sequence<5> (1..14)
            │                           ├── MultiplicativeExpr (1..2)
            │                           │   └── UnaryExpr (1..2)
            │                           │       └── PrimaryExpr (1..2)
            │                           │           └── Number (1..2)
            │                           │               └── "1" (1..2)
            │                           ├── "+" (3..4)
            │                           ├── MultiplicativeExpr (5..10)
            │                           │   └── Sequence<3> (5..10)
            │                           │       ├── UnaryExpr (5..6)
            │                           │       │   └── PrimaryExpr (5..6)
            │                           │       │       └── Number (5..6)
            │                           │       │           └── "2" (5..6)
            │                           │       ├── "*" (7..8)
            │                           │       └── UnaryExpr (9..10)
            │                           │           └── PrimaryExpr (9..10)
            │                           │               └── Number (9..10)
            │                           │                   └── "3" (9..10)
            │                           ├── "+" (11..12)
            │                           └── MultiplicativeExpr (13..14)
            │                               └── UnaryExpr (13..14)
            │                                   └── PrimaryExpr (13..14)
            │                                       └── Number (13..14)
            │                                           └── "4" (13..14)
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
            │   └── LogicalOrExpr (1..12)
            │       └── LogicalAndExpr (1..12)
            │           └── EqualityExpr (1..12)
            │               └── ComparisonExpr (1..12)
            │                   └── AdditiveExpr (1..12)
            │                       └── MultiplicativeExpr (1..12)
            │                           └── Sequence<3> (1..12)
            │                               ├── UnaryExpr (1..8)
            │                               │   └── PrimaryExpr (1..8)
            │                               │       └── Sequence<3> (1..8)
            │                               │           ├── "(" (1..2)
            │                               │           ├── Expr (2..7)
            │                               │           │   └── LogicalOrExpr (2..7)
            │                               │           │       └── LogicalAndExpr (2..7)
            │                               │           │           └── EqualityExpr (2..7)
            │                               │           │               └── ComparisonExpr (2..7)
            │                               │           │                   └── AdditiveExpr (2..7)
            │                               │           │                       └── Sequence<3> (2..7)
            │                               │           │                           ├── MultiplicativeExpr (2..3)
            │                               │           │                           │   └── UnaryExpr (2..3)
            │                               │           │                           │       └── PrimaryExpr (2..3)
            │                               │           │                           │           └── Number (2..3)
            │                               │           │                           │               └── "1" (2..3)
            │                               │           │                           ├── "+" (4..5)
            │                               │           │                           └── MultiplicativeExpr (6..7)
            │                               │           │                               └── UnaryExpr (6..7)
            │                               │           │                                   └── PrimaryExpr (6..7)
            │                               │           │                                       └── Number (6..7)
            │                               │           │                                           └── "2" (6..7)
            │                               │           └── ")" (7..8)
            │                               ├── "*" (9..10)
            │                               └── UnaryExpr (11..12)
            │                                   └── PrimaryExpr (11..12)
            │                                       └── Number (11..12)
            │                                           └── "3" (11..12)
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
            │   └── LogicalOrExpr (1..3)
            │       └── LogicalAndExpr (1..3)
            │           └── EqualityExpr (1..3)
            │               └── ComparisonExpr (1..3)
            │                   └── AdditiveExpr (1..3)
            │                       └── MultiplicativeExpr (1..3)
            │                           └── UnaryExpr (1..3)
            │                               └── Sequence<2> (1..3)
            │                                   ├── "-" (1..2)
            │                                   └── UnaryExpr (2..3)
            │                                       └── PrimaryExpr (2..3)
            │                                           └── Number (2..3)
            │                                               └── "1" (2..3)
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
            │   └── LogicalOrExpr (1..7)
            │       └── LogicalAndExpr (1..7)
            │           └── EqualityExpr (1..7)
            │               └── ComparisonExpr (1..7)
            │                   └── AdditiveExpr (1..7)
            │                       └── Sequence<3> (1..7)
            │                           ├── MultiplicativeExpr (1..3)
            │                           │   └── UnaryExpr (1..3)
            │                           │       └── Sequence<2> (1..3)
            │                           │           ├── "-" (1..2)
            │                           │           └── UnaryExpr (2..3)
            │                           │               └── PrimaryExpr (2..3)
            │                           │                   └── Number (2..3)
            │                           │                       └── "1" (2..3)
            │                           ├── "+" (4..5)
            │                           └── MultiplicativeExpr (6..7)
            │                               └── UnaryExpr (6..7)
            │                                   └── PrimaryExpr (6..7)
            │                                       └── Number (6..7)
            │                                           └── "2" (6..7)
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
            │   └── LogicalOrExpr (1..11)
            │       └── LogicalAndExpr (1..11)
            │           └── EqualityExpr (1..11)
            │               └── ComparisonExpr (1..11)
            │                   └── AdditiveExpr (1..11)
            │                       └── MultiplicativeExpr (1..11)
            │                           └── Sequence<5> (1..11)
            │                               ├── UnaryExpr (1..3)
            │                               │   └── PrimaryExpr (1..3)
            │                               │       └── Number (1..3)
            │                               │           └── "10" (1..3)
            │                               ├── "/" (4..5)
            │                               ├── UnaryExpr (6..7)
            │                               │   └── PrimaryExpr (6..7)
            │                               │       └── Number (6..7)
            │                               │           └── "2" (6..7)
            │                               ├── "%" (8..9)
            │                               └── UnaryExpr (10..11)
            │                                   └── PrimaryExpr (10..11)
            │                                       └── Number (10..11)
            │                                           └── "3" (10..11)
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
            │   └── LogicalOrExpr (1..6)
            │       └── LogicalAndExpr (1..6)
            │           └── EqualityExpr (1..6)
            │               └── ComparisonExpr (1..6)
            │                   └── Sequence<3> (1..6)
            │                       ├── AdditiveExpr (1..2)
            │                       │   └── MultiplicativeExpr (1..2)
            │                       │       └── UnaryExpr (1..2)
            │                       │           └── PrimaryExpr (1..2)
            │                       │               └── Number (1..2)
            │                       │                   └── "1" (1..2)
            │                       ├── "<" (3..4)
            │                       └── AdditiveExpr (5..6)
            │                           └── MultiplicativeExpr (5..6)
            │                               └── UnaryExpr (5..6)
            │                                   └── PrimaryExpr (5..6)
            │                                       └── Number (5..6)
            │                                           └── "2" (5..6)
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
            │   └── LogicalOrExpr (1..14)
            │       └── LogicalAndExpr (1..14)
            │           └── EqualityExpr (1..14)
            │               └── ComparisonExpr (1..14)
            │                   └── Sequence<3> (1..14)
            │                       ├── AdditiveExpr (1..6)
            │                       │   └── Sequence<3> (1..6)
            │                       │       ├── MultiplicativeExpr (1..2)
            │                       │       │   └── UnaryExpr (1..2)
            │                       │       │       └── PrimaryExpr (1..2)
            │                       │       │           └── Number (1..2)
            │                       │       │               └── "1" (1..2)
            │                       │       ├── "+" (3..4)
            │                       │       └── MultiplicativeExpr (5..6)
            │                       │           └── UnaryExpr (5..6)
            │                       │               └── PrimaryExpr (5..6)
            │                       │                   └── Number (5..6)
            │                       │                       └── "2" (5..6)
            │                       ├── "<" (7..8)
            │                       └── AdditiveExpr (9..14)
            │                           └── MultiplicativeExpr (9..14)
            │                               └── Sequence<3> (9..14)
            │                                   ├── UnaryExpr (9..10)
            │                                   │   └── PrimaryExpr (9..10)
            │                                   │       └── Number (9..10)
            │                                   │           └── "3" (9..10)
            │                                   ├── "*" (11..12)
            │                                   └── UnaryExpr (13..14)
            │                                       └── PrimaryExpr (13..14)
            │                                           └── Number (13..14)
            │                                               └── "4" (13..14)
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
            │   └── LogicalOrExpr (1..7)
            │       └── LogicalAndExpr (1..7)
            │           └── EqualityExpr (1..7)
            │               └── Sequence<3> (1..7)
            │                   ├── ComparisonExpr (1..2)
            │                   │   └── AdditiveExpr (1..2)
            │                   │       └── MultiplicativeExpr (1..2)
            │                   │           └── UnaryExpr (1..2)
            │                   │               └── PrimaryExpr (1..2)
            │                   │                   └── Identifier (1..2)
            │                   │                       └── "x" (1..2)
            │                   ├── "==" (3..5)
            │                   └── ComparisonExpr (6..7)
            │                       └── AdditiveExpr (6..7)
            │                           └── MultiplicativeExpr (6..7)
            │                               └── UnaryExpr (6..7)
            │                                   └── PrimaryExpr (6..7)
            │                                       └── Number (6..7)
            │                                           └── "0" (6..7)
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
            │   └── LogicalOrExpr (1..7)
            │       └── LogicalAndExpr (1..7)
            │           └── EqualityExpr (1..7)
            │               └── Sequence<3> (1..7)
            │                   ├── ComparisonExpr (1..2)
            │                   │   └── AdditiveExpr (1..2)
            │                   │       └── MultiplicativeExpr (1..2)
            │                   │           └── UnaryExpr (1..2)
            │                   │               └── PrimaryExpr (1..2)
            │                   │                   └── Identifier (1..2)
            │                   │                       └── "x" (1..2)
            │                   ├── "!=" (3..5)
            │                   └── ComparisonExpr (6..7)
            │                       └── AdditiveExpr (6..7)
            │                           └── MultiplicativeExpr (6..7)
            │                               └── UnaryExpr (6..7)
            │                                   └── PrimaryExpr (6..7)
            │                                       └── Identifier (6..7)
            │                                           └── "y" (6..7)
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
            │   └── LogicalOrExpr (1..15)
            │       └── LogicalAndExpr (1..15)
            │           └── Sequence<3> (1..15)
            │               ├── EqualityExpr (1..6)
            │               │   └── ComparisonExpr (1..6)
            │               │       └── Sequence<3> (1..6)
            │               │           ├── AdditiveExpr (1..2)
            │               │           │   └── MultiplicativeExpr (1..2)
            │               │           │       └── UnaryExpr (1..2)
            │               │           │           └── PrimaryExpr (1..2)
            │               │           │               └── Identifier (1..2)
            │               │           │                   └── "a" (1..2)
            │               │           ├── ">" (3..4)
            │               │           └── AdditiveExpr (5..6)
            │               │               └── MultiplicativeExpr (5..6)
            │               │                   └── UnaryExpr (5..6)
            │               │                       └── PrimaryExpr (5..6)
            │               │                           └── Number (5..6)
            │               │                               └── "0" (5..6)
            │               ├── "&&" (7..9)
            │               └── EqualityExpr (10..15)
            │                   └── ComparisonExpr (10..15)
            │                       └── Sequence<3> (10..15)
            │                           ├── AdditiveExpr (10..11)
            │                           │   └── MultiplicativeExpr (10..11)
            │                           │       └── UnaryExpr (10..11)
            │                           │           └── PrimaryExpr (10..11)
            │                           │               └── Identifier (10..11)
            │                           │                   └── "b" (10..11)
            │                           ├── ">" (12..13)
            │                           └── AdditiveExpr (14..15)
            │                               └── MultiplicativeExpr (14..15)
            │                                   └── UnaryExpr (14..15)
            │                                       └── PrimaryExpr (14..15)
            │                                           └── Number (14..15)
            │                                               └── "0" (14..15)
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
            │   └── LogicalOrExpr (1..17)
            │       └── Sequence<3> (1..17)
            │           ├── LogicalAndExpr (1..7)
            │           │   └── EqualityExpr (1..7)
            │           │       └── Sequence<3> (1..7)
            │           │           ├── ComparisonExpr (1..2)
            │           │           │   └── AdditiveExpr (1..2)
            │           │           │       └── MultiplicativeExpr (1..2)
            │           │           │           └── UnaryExpr (1..2)
            │           │           │               └── PrimaryExpr (1..2)
            │           │           │                   └── Identifier (1..2)
            │           │           │                       └── "x" (1..2)
            │           │           ├── "==" (3..5)
            │           │           └── ComparisonExpr (6..7)
            │           │               └── AdditiveExpr (6..7)
            │           │                   └── MultiplicativeExpr (6..7)
            │           │                       └── UnaryExpr (6..7)
            │           │                           └── PrimaryExpr (6..7)
            │           │                               └── Number (6..7)
            │           │                                   └── "1" (6..7)
            │           ├── "||" (8..10)
            │           └── LogicalAndExpr (11..17)
            │               └── EqualityExpr (11..17)
            │                   └── Sequence<3> (11..17)
            │                       ├── ComparisonExpr (11..12)
            │                       │   └── AdditiveExpr (11..12)
            │                       │       └── MultiplicativeExpr (11..12)
            │                       │           └── UnaryExpr (11..12)
            │                       │               └── PrimaryExpr (11..12)
            │                       │                   └── Identifier (11..12)
            │                       │                       └── "y" (11..12)
            │                       ├── "==" (13..15)
            │                       └── ComparisonExpr (16..17)
            │                           └── AdditiveExpr (16..17)
            │                               └── MultiplicativeExpr (16..17)
            │                                   └── UnaryExpr (16..17)
            │                                       └── PrimaryExpr (16..17)
            │                                           └── Number (16..17)
            │                                               └── "2" (16..17)
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
            │   └── LogicalOrExpr (1..29)
            │       └── Sequence<3> (1..29)
            │           ├── LogicalAndExpr (1..15)
            │           │   └── EqualityExpr (1..15)
            │           │       └── Sequence<3> (1..15)
            │           │           ├── ComparisonExpr (1..10)
            │           │           │   └── AdditiveExpr (1..10)
            │           │           │       └── Sequence<3> (1..10)
            │           │           │           ├── MultiplicativeExpr (1..2)
            │           │           │           │   └── UnaryExpr (1..2)
            │           │           │           │       └── PrimaryExpr (1..2)
            │           │           │           │           └── Identifier (1..2)
            │           │           │           │               └── "a" (1..2)
            │           │           │           ├── "+" (3..4)
            │           │           │           └── MultiplicativeExpr (5..10)
            │           │           │               └── Sequence<3> (5..10)
            │           │           │                   ├── UnaryExpr (5..6)
            │           │           │                   │   └── PrimaryExpr (5..6)
            │           │           │                   │       └── Identifier (5..6)
            │           │           │                   │           └── "b" (5..6)
            │           │           │                   ├── "*" (7..8)
            │           │           │                   └── UnaryExpr (9..10)
            │           │           │                       └── PrimaryExpr (9..10)
            │           │           │                           └── Identifier (9..10)
            │           │           │                               └── "c" (9..10)
            │           │           ├── "==" (11..13)
            │           │           └── ComparisonExpr (14..15)
            │           │               └── AdditiveExpr (14..15)
            │           │                   └── MultiplicativeExpr (14..15)
            │           │                       └── UnaryExpr (14..15)
            │           │                           └── PrimaryExpr (14..15)
            │           │                               └── Identifier (14..15)
            │           │                                   └── "d" (14..15)
            │           ├── "||" (16..18)
            │           └── LogicalAndExpr (19..29)
            │               └── Sequence<3> (19..29)
            │                   ├── EqualityExpr (19..20)
            │                   │   └── ComparisonExpr (19..20)
            │                   │       └── AdditiveExpr (19..20)
            │                   │           └── MultiplicativeExpr (19..20)
            │                   │               └── UnaryExpr (19..20)
            │                   │                   └── PrimaryExpr (19..20)
            │                   │                       └── Identifier (19..20)
            │                   │                           └── "e" (19..20)
            │                   ├── "&&" (21..23)
            │                   └── EqualityExpr (24..29)
            │                       └── ComparisonExpr (24..29)
            │                           └── Sequence<3> (24..29)
            │                               ├── AdditiveExpr (24..25)
            │                               │   └── MultiplicativeExpr (24..25)
            │                               │       └── UnaryExpr (24..25)
            │                               │           └── PrimaryExpr (24..25)
            │                               │               └── Identifier (24..25)
            │                               │                   └── "f" (24..25)
            │                               ├── "<" (26..27)
            │                               └── AdditiveExpr (28..29)
            │                                   └── MultiplicativeExpr (28..29)
            │                                       └── UnaryExpr (28..29)
            │                                           └── PrimaryExpr (28..29)
            │                                               └── Identifier (28..29)
            │                                                   └── "g" (28..29)
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
            │   └── LogicalOrExpr (1..8)
            │       └── LogicalAndExpr (1..8)
            │           └── EqualityExpr (1..8)
            │               └── ComparisonExpr (1..8)
            │                   └── Sequence<3> (1..8)
            │                       ├── AdditiveExpr (1..2)
            │                       │   └── MultiplicativeExpr (1..2)
            │                       │       └── UnaryExpr (1..2)
            │                       │           └── PrimaryExpr (1..2)
            │                       │               └── Identifier (1..2)
            │                       │                   └── "x" (1..2)
            │                       ├── "<=" (3..5)
            │                       └── AdditiveExpr (6..8)
            │                           └── MultiplicativeExpr (6..8)
            │                               └── UnaryExpr (6..8)
            │                                   └── PrimaryExpr (6..8)
            │                                       └── Number (6..8)
            │                                           └── "10" (6..8)
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
            │   └── LogicalOrExpr (1..7)
            │       └── LogicalAndExpr (1..7)
            │           └── EqualityExpr (1..7)
            │               └── ComparisonExpr (1..7)
            │                   └── Sequence<3> (1..7)
            │                       ├── AdditiveExpr (1..2)
            │                       │   └── MultiplicativeExpr (1..2)
            │                       │       └── UnaryExpr (1..2)
            │                       │           └── PrimaryExpr (1..2)
            │                       │               └── Identifier (1..2)
            │                       │                   └── "x" (1..2)
            │                       ├── ">=" (3..5)
            │                       └── AdditiveExpr (6..7)
            │                           └── MultiplicativeExpr (6..7)
            │                               └── UnaryExpr (6..7)
            │                                   └── PrimaryExpr (6..7)
            │                                       └── Number (6..7)
            │                                           └── "0" (6..7)
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
            │   └── LogicalOrExpr (1..10)
            │       └── LogicalAndExpr (1..10)
            │           └── EqualityExpr (1..10)
            │               └── ComparisonExpr (1..10)
            │                   └── AdditiveExpr (1..10)
            │                       └── MultiplicativeExpr (1..10)
            │                           └── UnaryExpr (1..10)
            │                               └── PrimaryExpr (1..10)
            │                                   └── Sequence<3> (1..10)
            │                                       ├── "(" (1..2)
            │                                       ├── Expr (2..9)
            │                                       │   └── LogicalOrExpr (2..9)
            │                                       │       └── LogicalAndExpr (2..9)
            │                                       │           └── EqualityExpr (2..9)
            │                                       │               └── ComparisonExpr (2..9)
            │                                       │                   └── AdditiveExpr (2..9)
            │                                       │                       └── MultiplicativeExpr (2..9)
            │                                       │                           └── UnaryExpr (2..9)
            │                                       │                               └── PrimaryExpr (2..9)
            │                                       │                                   └── Sequence<3> (2..9)
            │                                       │                                       ├── "(" (2..3)
            │                                       │                                       ├── Expr (3..8)
            │                                       │                                       │   └── LogicalOrExpr (3..8)
            │                                       │                                       │       └── LogicalAndExpr (3..8)
            │                                       │                                       │           └── EqualityExpr (3..8)
            │                                       │                                       │               └── ComparisonExpr (3..8)
            │                                       │                                       │                   └── AdditiveExpr (3..8)
            │                                       │                                       │                       └── Sequence<3> (3..8)
            │                                       │                                       │                           ├── MultiplicativeExpr (3..4)
            │                                       │                                       │                           │   └── UnaryExpr (3..4)
            │                                       │                                       │                           │       └── PrimaryExpr (3..4)
            │                                       │                                       │                           │           └── Number (3..4)
            │                                       │                                       │                           │               └── "1" (3..4)
            │                                       │                                       │                           ├── "+" (5..6)
            │                                       │                                       │                           └── MultiplicativeExpr (7..8)
            │                                       │                                       │                               └── UnaryExpr (7..8)
            │                                       │                                       │                                   └── PrimaryExpr (7..8)
            │                                       │                                       │                                       └── Number (7..8)
            │                                       │                                       │                                           └── "2" (7..8)
            │                                       │                                       └── ")" (8..9)
            │                                       └── ")" (9..10)
            └── ";" (10..11)`,
		},
		{
			Name:  "Identifier expression",
			Input: "foo;",
			Expected: `Program (1..5)
└── Stmt (1..5)
    └── ExprStmt (1..5)
        └── Sequence<2> (1..5)
            ├── Expr (1..4)
            │   └── LogicalOrExpr (1..4)
            │       └── LogicalAndExpr (1..4)
            │           └── EqualityExpr (1..4)
            │               └── ComparisonExpr (1..4)
            │                   └── AdditiveExpr (1..4)
            │                       └── MultiplicativeExpr (1..4)
            │                           └── UnaryExpr (1..4)
            │                               └── PrimaryExpr (1..4)
            │                                   └── Identifier (1..4)
            │                                       └── "foo" (1..4)
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
            │   └── LogicalOrExpr (1..2)
            │       └── LogicalAndExpr (1..2)
            │           └── EqualityExpr (1..2)
            │               └── ComparisonExpr (1..2)
            │                   └── AdditiveExpr (1..2)
            │                       └── MultiplicativeExpr (1..2)
            │                           └── UnaryExpr (1..2)
            │                               └── PrimaryExpr (1..2)
            │                                   └── Number (1..2)
            │                                       └── "0" (1..2)
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
            │   └── LogicalOrExpr (9..10)
            │       └── LogicalAndExpr (9..10)
            │           └── EqualityExpr (9..10)
            │               └── ComparisonExpr (9..10)
            │                   └── AdditiveExpr (9..10)
            │                       └── MultiplicativeExpr (9..10)
            │                           └── UnaryExpr (9..10)
            │                               └── PrimaryExpr (9..10)
            │                                   └── Number (9..10)
            │                                       └── "1" (9..10)
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
            │   └── LogicalOrExpr (1..3)
            │       └── LogicalAndExpr (1..3)
            │           └── EqualityExpr (1..3)
            │               └── ComparisonExpr (1..3)
            │                   └── AdditiveExpr (1..3)
            │                       └── MultiplicativeExpr (1..3)
            │                           └── UnaryExpr (1..3)
            │                               └── PrimaryExpr (1..3)
            │                                   └── Number (1..3)
            │                                       └── "42" (1..3)
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
            │   └── LogicalOrExpr (1..7)
            │       └── LogicalAndExpr (1..7)
            │           └── EqualityExpr (1..7)
            │               └── ComparisonExpr (1..7)
            │                   └── AdditiveExpr (1..7)
            │                       └── MultiplicativeExpr (1..7)
            │                           └── UnaryExpr (1..7)
            │                               └── PrimaryExpr (1..7)
            │                                   └── Sequence<3> (1..7)
            │                                       ├── "(" (1..2)
            │                                       ├── Expr (2..7)
            │                                       │   └── LogicalOrExpr (2..7)
            │                                       │       └── LogicalAndExpr (2..7)
            │                                       │           └── EqualityExpr (2..7)
            │                                       │               └── ComparisonExpr (2..7)
            │                                       │                   └── AdditiveExpr (2..7)
            │                                       │                       └── Sequence<3> (2..7)
            │                                       │                           ├── MultiplicativeExpr (2..3)
            │                                       │                           │   └── UnaryExpr (2..3)
            │                                       │                           │       └── PrimaryExpr (2..3)
            │                                       │                           │           └── Number (2..3)
            │                                       │                           │               └── "1" (2..3)
            │                                       │                           ├── "+" (4..5)
            │                                       │                           └── MultiplicativeExpr (6..7)
            │                                       │                               └── UnaryExpr (6..7)
            │                                       │                                   └── PrimaryExpr (6..7)
            │                                       │                                       └── Number (6..7)
            │                                       │                                           └── "2" (6..7)
            │                                       └── Error<closeparen> (7)
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
            │   └── LogicalOrExpr (5..10)
            │       └── LogicalAndExpr (5..10)
            │           └── EqualityExpr (5..10)
            │               └── ComparisonExpr (5..10)
            │                   └── Sequence<3> (5..10)
            │                       ├── AdditiveExpr (5..6)
            │                       │   └── MultiplicativeExpr (5..6)
            │                       │       └── UnaryExpr (5..6)
            │                       │           └── PrimaryExpr (5..6)
            │                       │               └── Identifier (5..6)
            │                       │                   └── "x" (5..6)
            │                       ├── ">" (7..8)
            │                       └── AdditiveExpr (9..10)
            │                           └── MultiplicativeExpr (9..10)
            │                               └── UnaryExpr (9..10)
            │                                   └── PrimaryExpr (9..10)
            │                                       └── Number (9..10)
            │                                           └── "0" (9..10)
            ├── ")" (10..11)
            ├── "{" (12..13)
            ├── Stmt (14..16)
            │   └── ExprStmt (14..16)
            │       └── Sequence<2> (14..16)
            │           ├── Expr (14..15)
            │           │   └── LogicalOrExpr (14..15)
            │           │       └── LogicalAndExpr (14..15)
            │           │           └── EqualityExpr (14..15)
            │           │               └── ComparisonExpr (14..15)
            │           │                   └── AdditiveExpr (14..15)
            │           │                       └── MultiplicativeExpr (14..15)
            │           │                           └── UnaryExpr (14..15)
            │           │                               └── PrimaryExpr (14..15)
            │           │                                   └── Identifier (14..15)
            │           │                                       └── "x" (14..15)
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
            │   └── LogicalOrExpr (4..9)
            │       └── LogicalAndExpr (4..9)
            │           └── EqualityExpr (4..9)
            │               └── ComparisonExpr (4..9)
            │                   └── Sequence<3> (4..9)
            │                       ├── AdditiveExpr (4..5)
            │                       │   └── MultiplicativeExpr (4..5)
            │                       │       └── UnaryExpr (4..5)
            │                       │           └── PrimaryExpr (4..5)
            │                       │               └── Identifier (4..5)
            │                       │                   └── "x" (4..5)
            │                       ├── ">" (6..7)
            │                       └── AdditiveExpr (8..9)
            │                           └── MultiplicativeExpr (8..9)
            │                               └── UnaryExpr (8..9)
            │                                   └── PrimaryExpr (8..9)
            │                                       └── Number (8..9)
            │                                           └── "0" (8..9)
            ├── ")" (9..10)
            ├── "{" (11..12)
            ├── Stmt (13..15)
            │   └── ExprStmt (13..15)
            │       └── Sequence<2> (13..15)
            │           ├── Expr (13..14)
            │           │   └── LogicalOrExpr (13..14)
            │           │       └── LogicalAndExpr (13..14)
            │           │           └── EqualityExpr (13..14)
            │           │               └── ComparisonExpr (13..14)
            │           │                   └── AdditiveExpr (13..14)
            │           │                       └── MultiplicativeExpr (13..14)
            │           │                           └── UnaryExpr (13..14)
            │           │                               └── PrimaryExpr (13..14)
            │           │                                   └── Identifier (13..14)
            │           │                                       └── "x" (13..14)
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

// Benchmark scenarios of increasing complexity to stress the
// non-left-recursive grammar with operator precedence and recovery.
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
