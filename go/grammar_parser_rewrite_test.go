package langlang

import "testing"

// testTree is a helper to build trees for testing without worrying
// about byte offsets. All text content goes through named string nodes.
type testTree struct {
	tree *tree
}

func newTestTree() *testTree {
	return &testTree{
		tree: &tree{
			strs:  []string{},
			input: []byte{},
		},
	}
}

func (tt *testTree) intern(s string) int32 {
	for i, existing := range tt.tree.strs {
		if existing == s {
			return int32(i)
		}
	}
	id := int32(len(tt.tree.strs))
	tt.tree.strs = append(tt.tree.strs, s)
	return id
}

func (tt *testTree) namedStr(name, text string) NodeID {
	nameID := tt.intern(name)
	// Put the text into the input buffer so Text() works
	start := len(tt.tree.input)
	tt.tree.input = append(tt.tree.input, []byte(text)...)
	end := len(tt.tree.input)
	strID := tt.tree.AddString(start, end)
	return tt.tree.AddNode(nameID, strID, start, end)
}

func (tt *testTree) named(name string, children ...NodeID) NodeID {
	nameID := tt.intern(name)
	if len(children) == 1 {
		return tt.tree.AddNode(nameID, children[0], 0, 0)
	}
	seq := tt.tree.AddSequence(children, 0, 0)
	return tt.tree.AddNode(nameID, seq, 0, 0)
}

func TestParseTypeDeclFromTree(t *testing.T) {
	tt := newTestTree()

	exprName := tt.namedStr("Identifier", "Expr")

	// Binary(Op: string, Left: Expr)
	opName := tt.namedStr("Identifier", "Op")
	opType := tt.namedStr("TypeRef", "string")
	opField := tt.named("FieldDecl", opName, opType)

	leftName := tt.namedStr("Identifier", "Left")
	leftType := tt.namedStr("TypeRef", "Expr")
	leftField := tt.named("FieldDecl", leftName, leftType)

	binaryFieldList := tt.named("FieldList", opField, leftField)
	binaryName := tt.namedStr("Identifier", "Binary")
	binaryDecl := tt.named("CtorDecl", binaryName, binaryFieldList)

	// Num(Value: string)
	valueName := tt.namedStr("Identifier", "Value")
	valueType := tt.namedStr("TypeRef", "string")
	valueField := tt.named("FieldDecl", valueName, valueType)
	numFieldList := tt.named("FieldList", valueField)
	numName := tt.namedStr("Identifier", "Num")
	numDecl := tt.named("CtorDecl", numName, numFieldList)

	ctorList := tt.named("CtorList", binaryDecl, numDecl)
	typeDecl := tt.named("TypeDecl", exprName, ctorList)

	typ, err := ParseTypeDecl(tt.tree, typeDecl)
	if err != nil {
		t.Fatalf("ParseTypeDecl failed: %v", err)
	}

	if typ.Name != "Expr" {
		t.Fatalf("expected type name 'Expr', got %q", typ.Name)
	}
	if len(typ.Constructors) != 2 {
		t.Fatalf("expected 2 constructors, got %d", len(typ.Constructors))
	}
	if typ.Constructors[0].Name != "Binary" {
		t.Fatalf("expected first ctor 'Binary', got %q", typ.Constructors[0].Name)
	}
	if len(typ.Constructors[0].Fields) != 2 {
		t.Fatalf("expected 2 fields for Binary, got %d", len(typ.Constructors[0].Fields))
	}
	if typ.Constructors[1].Name != "Num" {
		t.Fatalf("expected second ctor 'Num', got %q", typ.Constructors[1].Name)
	}
	if typ.Constructors[1].Fields[0].Name != "Value" {
		t.Fatalf("expected Num field 'Value', got %q", typ.Constructors[1].Fields[0].Name)
	}
}
