package langlang

import (
	"strings"
	"testing"
)

func TestGenGoTypes(t *testing.T) {
	types := []*TypeDecl{
		{
			Name: "Expr",
			Constructors: []*Constructor{
				{Name: "Binary", Fields: []*Field{
					{Name: "Op", TypeName: "string"},
					{Name: "Left", TypeName: "Expr"},
					{Name: "Right", TypeName: "Expr"},
				}},
				{Name: "NumLit", Fields: []*Field{
					{Name: "Value", TypeName: "string"},
				}},
				{Name: "Ident", Fields: []*Field{
					{Name: "Name", TypeName: "string"},
				}},
			},
		},
	}

	src, err := GenGoTypes(types, GenGoRewriteOptions{PackageName: "ast"})
	if err != nil {
		t.Fatalf("GenGoTypes failed: %v", err)
	}

	// Verify key elements are present (gofmt may add tab-aligned spacing)
	for _, want := range []string{
		"package ast",
		"type Expr interface",
		"type Binary struct",
		"Op",
		"Left",
		"Right",
		"type NumLit struct",
		"Value",
		"type Ident struct",
		"func (*Binary) expr()",
		"func (*NumLit) expr()",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q", want)
		}
	}

	t.Logf("Generated code:\n%s", src)
}

func TestGenGoRewriteFunctions(t *testing.T) {
	rules := []*RewriteRuleSet{
		{
			Name: "expr",
			Rules: []*RewriteRule{
				{
					Name: "expr",
					Pattern: PatNamed{
						NodeName: "Binary",
						Body: PatSeq{Elems: []RewritePattern{
							PatVar{Name: "op"},
							PatVar{Name: "l"},
							PatVar{Name: "r"},
						}},
					},
					Constr: ConNamed{NodeName: "BinOp", Body: ConVar{Name: "op"}},
				},
				{
					Name: "expr",
					Pattern: PatNamed{
						NodeName: "NumLit",
						Body:     PatVar{Name: "v"},
					},
					Constr: ConNamed{NodeName: "Push", Body: ConVar{Name: "v"}},
				},
			},
		},
	}

	src, err := GenGoRewriteFunctions(rules, GenGoRewriteOptions{PackageName: "codegen"})
	if err != nil {
		t.Fatalf("GenGoRewriteFunctions failed: %v", err)
	}

	for _, want := range []string{
		"package codegen",
		"func (rw *Rewriter) expr",
		"switch name",
		`case "Binary"`,
		`case "NumLit"`,
		"non-exhaustive",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing %q", want)
		}
	}

	t.Logf("Generated code:\n%s", src)
}

func TestGenGoTypesListField(t *testing.T) {
	types := []*TypeDecl{
		{
			Name: "Stmt",
			Constructors: []*Constructor{
				{Name: "Block", Fields: []*Field{
					{Name: "Items", TypeName: "list[Stmt]"},
				}},
			},
		},
	}

	src, err := GenGoTypes(types, GenGoRewriteOptions{PackageName: "ast"})
	if err != nil {
		t.Fatalf("GenGoTypes failed: %v", err)
	}

	if !strings.Contains(src, "[]Stmt") {
		t.Errorf("expected []Stmt for list[Stmt], got:\n%s", src)
	}
}
