package main

import "go/ast"

// OracleFlavor generates a prefix-matching VM for grammar based
// constrained decoding
//
// Main difference from the regular MatchRule is that EOF (input
// exhaustion) returns success, not failure.
var OracleFlavor = VMFlavor{
	Name:            "oracle",
	FuncName:        "advancePrefix",
	OutputFile:      "vm_oracle.go",
	MkSuccessReturn: mkOracleSuccessReturn,
	MkFailReturn:    mkOracleFailReturn,
	MkEOFReturn:     mkOracleEOFReturn,
	MkThrowReturn:   mkOracleFailReturn,
	Features:        VMFeatures{EOFReturnsSuccess: true, SpanStaysAtEOF: true},
	Params: []Field{
		{Names: []string{"data"}, Type: "[]byte"},
		{Names: []string{"pc", "cursor"}, Type: "int"},
	},
	Returns: []Field{
		{Type: "int"},  // finalPC
		{Type: "int"},  // finalCursor
		{Type: "bool"}, // ok
	},
}

// mkOracleSuccessReturn creates: return pc, cursor, stack, true
func mkOracleSuccessReturn() *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			ast.NewIdent("pc"),
			ast.NewIdent("cursor"),
			ast.NewIdent("true"),
		},
	}
}

// mkOracleFailReturn creates: return 0, 0, nil, false
func mkOracleFailReturn() *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			ast.NewIdent("0"),
			ast.NewIdent("0"),
			ast.NewIdent("false"),
		},
	}
}

// mkOracleEOFReturn creates: return pc, cursor, stack, true
// (Same as success - input exhaustion is success for prefix matching)
func mkOracleEOFReturn() *ast.ReturnStmt {
	return mkOracleSuccessReturn()
}
