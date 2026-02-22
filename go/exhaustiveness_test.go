package langlang

import "testing"

func TestExhaustivenessAllCovered(t *testing.T) {
	typ := &TypeDecl{
		Name: "Expr",
		Constructors: []*Constructor{
			{Name: "Add", Fields: []*Field{
				{Name: "Left", TypeName: "Expr"},
				{Name: "Right", TypeName: "Expr"},
			}},
			{Name: "Num", Fields: []*Field{
				{Name: "Value", TypeName: "string"},
			}},
		},
	}

	patterns := []RewritePattern{
		PatNamed{NodeName: "Add", Body: PatSeq{Elems: []RewritePattern{
			PatVar{Name: "l"}, PatVar{Name: "r"},
		}}},
		PatNamed{NodeName: "Num", Body: PatVar{Name: "v"}},
	}

	uncovered := CheckExhaustiveness(typ, patterns)
	if len(uncovered) != 0 {
		t.Fatalf("expected exhaustive, got uncovered: %v", uncovered)
	}
}

func TestExhaustivenessMissing(t *testing.T) {
	typ := &TypeDecl{
		Name: "Expr",
		Constructors: []*Constructor{
			{Name: "Add", Fields: []*Field{
				{Name: "Left", TypeName: "Expr"},
				{Name: "Right", TypeName: "Expr"},
			}},
			{Name: "Mul", Fields: []*Field{
				{Name: "Left", TypeName: "Expr"},
				{Name: "Right", TypeName: "Expr"},
			}},
			{Name: "Num", Fields: []*Field{
				{Name: "Value", TypeName: "string"},
			}},
			{Name: "Ident", Fields: []*Field{
				{Name: "Name", TypeName: "string"},
			}},
		},
	}

	// Only covering Add and Num
	patterns := []RewritePattern{
		PatNamed{NodeName: "Add", Body: PatSeq{Elems: []RewritePattern{
			PatVar{Name: "l"}, PatVar{Name: "r"},
		}}},
		PatNamed{NodeName: "Num", Body: PatVar{Name: "v"}},
	}

	uncovered := CheckExhaustiveness(typ, patterns)
	if len(uncovered) != 2 {
		t.Fatalf("expected 2 uncovered, got %d: %v", len(uncovered), uncovered)
	}
	if uncovered[0].Constructor != "Mul" || uncovered[1].Constructor != "Ident" {
		t.Fatalf("expected Mul and Ident uncovered, got %v", uncovered)
	}
}

func TestExhaustivenessWildcard(t *testing.T) {
	typ := &TypeDecl{
		Name: "Expr",
		Constructors: []*Constructor{
			{Name: "Add", Fields: nil},
			{Name: "Num", Fields: nil},
		},
	}

	// A wildcard covers everything
	patterns := []RewritePattern{PatWild{}}
	uncovered := CheckExhaustiveness(typ, patterns)
	if len(uncovered) != 0 {
		t.Fatalf("wildcard should cover everything, got uncovered: %v", uncovered)
	}
}

func TestExhaustivenessVariable(t *testing.T) {
	typ := &TypeDecl{
		Name: "Expr",
		Constructors: []*Constructor{
			{Name: "Add", Fields: nil},
			{Name: "Num", Fields: nil},
		},
	}

	// A variable covers everything
	patterns := []RewritePattern{PatVar{Name: "x"}}
	uncovered := CheckExhaustiveness(typ, patterns)
	if len(uncovered) != 0 {
		t.Fatalf("variable should cover everything, got uncovered: %v", uncovered)
	}
}

func TestFormatExhaustivenessWarning(t *testing.T) {
	uncovered := []UncoveredPattern{
		{Constructor: "Mul", Fields: []string{"Left: Expr", "Right: Expr"}},
		{Constructor: "Ident", Fields: []string{"Name: string"}},
	}
	msg := FormatExhaustivenessWarning("expr", "Expr", uncovered)
	if msg == "" {
		t.Fatal("expected non-empty warning")
	}
	t.Log(msg)
}
