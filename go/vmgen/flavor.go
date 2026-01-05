package main

import "go/ast"

// VMFlavor defines the configuration for generating a VM variant.
// Each flavor produces a different function derived from MatchRule,
// with different behavior and injection points.
type VMFlavor struct {
	// Identity
	Name       string // "debug", "oracle"
	FuncName   string // "step", "advancePrefix"
	OutputFile string // "vm_debug.go", "vm_oracle.go"

	// Signature (receiver is inherited from MatchRule)
	Params  []Field // function parameters
	Returns []Field // return types

	// Behavioral features (toggle built-in transforms)
	Features VMFeatures

	// Code injection (nil = no injection)
	PreOpcodeHook  func() ast.Stmt // injected at start of main loop iteration
	PostOpcodeHook func() ast.Stmt // injected at end of main loop iteration

	// Return statement builders (since each mode returns differently)
	MkSuccessReturn func() *ast.ReturnStmt // for opHalt
	MkFailReturn    func() *ast.ReturnStmt // for fail path (after fail label)
	MkEOFReturn     func() *ast.ReturnStmt // for input exhaustion (if EOFReturnsSuccess)
	MkThrowReturn   func() *ast.ReturnStmt // for opThrow (nil = use MkFailReturn)

	// Escape hatch: additional transforms after built-ins
	ExtraTransforms []func(*ast.FuncDecl)
}

// Field represents a function parameter or return type.
type Field struct {
	Names []string // e.g., ["pc", "cursor"] or nil for unnamed returns
	Type  string   // e.g., "int", "[]byte", "*vmDebugSession"
}

// VMFeatures controls which built-in transforms are applied.
type VMFeatures struct {
	// DebugHooks injects breakpoint and step hooks
	// (PreOpcodeHook/PostOpcodeHook)
	DebugHooks bool

	// EOFReturnsSuccess transforms `if cursor >= ilen { goto fail }` into
	// a success return instead.  Used for prefix/oracle matching.
	EOFReturnsSuccess bool

	// SpanStaysAtEOF modifies opSpan to track whether any characters
	// matched, and if EOF is hit during matching, returns success with
	// PC staying at the span instruction. This allows NextChars to
	// include the span's charset.
	SpanStaysAtEOF bool
}

// ToASTFields converts a slice of Field to AST field declarations.
func ToASTFields(fields []Field) []*ast.Field {
	result := make([]*ast.Field, len(fields))
	for i, f := range fields {
		var names []*ast.Ident
		for _, n := range f.Names {
			names = append(names, ast.NewIdent(n))
		}
		result[i] = &ast.Field{
			Names: names,
			Type:  &ast.Ident{Name: f.Type},
		}
	}
	return result
}
