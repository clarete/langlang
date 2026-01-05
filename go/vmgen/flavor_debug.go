package main

import (
	"go/ast"
	"go/token"
)

// DebugFlavor generates a stepping debugger VM.
var DebugFlavor = VMFlavor{
	Name:            "debug",
	FuncName:        "step",
	OutputFile:      "vm_debug.go",
	Features:        VMFeatures{DebugHooks: true},
	PreOpcodeHook:   mkDebugPreOpcodeHook,
	PostOpcodeHook:  mkDebugPostOpcodeHook,
	MkSuccessReturn: mkDebugSuccessReturn,
	MkFailReturn:    mkDebugFailReturn,
	MkThrowReturn:   mkDebugSuccessReturn,
	Params: []Field{
		{Names: []string{"data"}, Type: "[]byte"},
		{Names: []string{"pc", "cursor"}, Type: "int"},
		{Names: []string{"hooks"}, Type: "*vmDebugSession"},
	},
	Returns: []Field{
		{Type: "int"},   // pc
		{Type: "int"},   // cursor
		{Type: "bool"},  // stopped
		{Type: "error"}, // error
	},
}

// mkDebugPreOpcodeHook creates the breakpoint check:
//
//	if hooks.IsBreakpoint(pc) {
//	    return hooks.Stop(StopBreakpoint, pc, cursor)
//	}
func mkDebugPreOpcodeHook() ast.Stmt {
	return &ast.IfStmt{
		Cond: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "hooks.IsBreakpoint"},
			Args: []ast.Expr{ast.NewIdent("pc")},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.Ident{Name: "hooks.Stop"},
							Args: []ast.Expr{
								ast.NewIdent("StopBreakpoint"),
								ast.NewIdent("pc"),
								ast.NewIdent("cursor"),
							},
						},
					},
				},
			},
		},
	}
}

// mkDebugPostOpcodeHook creates the step-stop check:
//
//	if hooks.ShouldStopAfter(op, pc, cursor) {
//	    return hooks.Stop(StopStep, pc, cursor)
//	}
func mkDebugPostOpcodeHook() ast.Stmt {
	return &ast.IfStmt{
		Cond: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "hooks.ShouldStopAfter"},
			Args: []ast.Expr{ast.NewIdent("op"), ast.NewIdent("pc"), ast.NewIdent("cursor")},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.Ident{Name: "hooks.Stop"},
							Args: []ast.Expr{
								ast.NewIdent("StopStep"),
								ast.NewIdent("pc"),
								ast.NewIdent("cursor"),
							},
						},
					},
				},
			},
		},
	}
}

// mkDebugSuccessReturn creates: return pc, cursor, false, nil
func mkDebugSuccessReturn() *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			ast.NewIdent("pc"),
			ast.NewIdent("cursor"),
			ast.NewIdent("false"),
			ast.NewIdent("nil"),
		},
	}
}

// mkDebugFailReturn creates: return pc, cursor, false, vm.mkErr(data, 0, cursor, vm.ffp)
func mkDebugFailReturn() *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			ast.NewIdent("pc"),
			ast.NewIdent("cursor"),
			ast.NewIdent("false"),
			&ast.CallExpr{
				Fun: &ast.Ident{Name: "vm.mkErr"},
				Args: []ast.Expr{
					ast.NewIdent("data"),
					&ast.BasicLit{Kind: token.INT, Value: "0"},
					ast.NewIdent("cursor"),
					&ast.Ident{Name: "vm.ffp"},
				},
			},
		},
	}
}
