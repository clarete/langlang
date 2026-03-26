package extract

import (
	"strings"
	"testing"
)

func TestEmitViewLeaf(t *testing.T) {
	rules := map[string]RuleInfo{
		"Ident": {Name: "Ident", Kind: RuleLeaf, NameID: 0},
	}

	code := emitViewTypes(rules, "Ident")

	checks := []string{
		"type Ident struct",
		"t *tree",
		"id NodeID",
		"func (v Ident) String() string",
		"v.t.Text(v.id)",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewChoice(t *testing.T) {
	rules := map[string]RuleInfo{
		"Value": {Name: "Value", Kind: RuleChoice, NameID: 0,
			Choices: []string{"Object", "String"}},
		"Object": {Name: "Object", Kind: RuleSequence, NameID: 1},
		"String": {Name: "String", Kind: RuleLeaf, NameID: 2},
	}

	code := emitViewTypes(rules, "Value")

	checks := []string{
		"type Value struct",
		"func (v Value) Object() (Object, bool)",
		"func (v Value) StringNode() (string, bool)",
		"_nameID_Object",
		"_nameID_String",
		"t.IsNamed(child,",
		"Text",
		// Choice for sequence child should use constructor
		"newObject(v.t, child)",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewChoiceLiterals(t *testing.T) {
	rules := map[string]RuleInfo{
		"Value": {Name: "Value", Kind: RuleChoice, NameID: 0,
			Choices: []string{"Object", "lit:true", "lit:false", "lit:null"}},
		"Object": {Name: "Object", Kind: RuleSequence, NameID: 1},
	}

	code := emitViewTypes(rules, "Value")

	checks := []string{
		"func (v Value) IsTrue() bool",
		"func (v Value) IsFalse() bool",
		"func (v Value) IsNull() bool",
		"NodeType_String",
		`Text(child) == "true"`,
		`Text(child) == "false"`,
		`Text(child) == "null"`,
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewSequence(t *testing.T) {
	rules := map[string]RuleInfo{
		"Member": {Name: "Member", Kind: RuleSequence, NameID: 0,
			Children: []RuleChild{
				{RuleName: "Key", Index: 0},
				{IsLiteral: true, Index: 1},
				{RuleName: "Value", Index: 2},
			}},
		"Key":   {Name: "Key", Kind: RuleLeaf, NameID: 1},
		"Value": {Name: "Value", Kind: RuleChoice, NameID: 2},
	}

	code := emitViewTypes(rules, "Member")

	checks := []string{
		"type Member struct",
		// Pre-resolved fields
		"_key NodeID",
		"_hasKey bool",
		"_value NodeID",
		"_hasValue bool",
		// Constructor
		"func newMember(t *tree, id NodeID) Member",
		"t.childRanges",
		"t.children[i]",
		"case _nameID_Key:",
		"case _nameID_Value:",
		// Leaf accessor is O(1)
		"func (v Member) Key() string",
		"v._hasKey",
		"v.t.Text(v._key)",
		// Non-leaf accessor is O(1)
		"func (v Member) Value() (Value, bool)",
		"v._hasValue",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewSequenceRepeated(t *testing.T) {
	rules := map[string]RuleInfo{
		"Array": {Name: "Array", Kind: RuleSequence, NameID: 0,
			Children: []RuleChild{
				{RuleName: "Value", Index: 0},
				{RuleName: "Value", Index: 1},
			}},
		"Value": {Name: "Value", Kind: RuleChoice, NameID: 1},
	}

	code := emitViewTypes(rules, "Array")

	checks := []string{
		"_value []NodeID",
		"func (v Array) ValueCount() int",
		"func (v Array) ValueAt(i int) Value",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewRepeat(t *testing.T) {
	rules := map[string]RuleInfo{
		"Items": {Name: "Items", Kind: RuleRepeat, NameID: 0, Inner: "Item"},
		"Item":  {Name: "Item", Kind: RuleSequence, NameID: 1},
	}

	code := emitViewTypes(rules, "Items")

	checks := []string{
		"type Items struct",
		"func (v Items) VisitItem(fn func(Item) bool)",
		"_nameID_Item",
		// Direct iteration, no Visit
		"t.childRanges",
		"t.children[i]",
		// Sequence child uses constructor
		"newItem(v.t, cid)",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewAlias(t *testing.T) {
	rules := map[string]RuleInfo{
		"Expr": {Name: "Expr", Kind: RuleAlias, NameID: 0, Inner: "Term"},
		"Term": {Name: "Term", Kind: RuleSequence, NameID: 1},
	}

	code := emitViewTypes(rules, "Expr")

	checks := []string{
		"type Expr struct",
		"func (v Expr) Term() (Term, bool)",
		// Sequence child uses constructor
		"newTerm(v.t, child)",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewOptional(t *testing.T) {
	rules := map[string]RuleInfo{
		"MaybeVal": {Name: "MaybeVal", Kind: RuleOptional, NameID: 0, Inner: "Val"},
		"Val":      {Name: "Val", Kind: RuleSequence, NameID: 1},
	}

	code := emitViewTypes(rules, "MaybeVal")

	checks := []string{
		"type MaybeVal struct",
		"func (v MaybeVal) Val() (Val, bool)",
	}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("missing %q in:\n%s", c, code)
		}
	}
}

func TestEmitViewSkipsNegativeNameID(t *testing.T) {
	rules := map[string]RuleInfo{
		"Recovery": {Name: "Recovery", Kind: RuleLeaf, NameID: -1},
	}

	code := emitViewTypes(rules, "")
	if strings.Contains(code, "Recovery") {
		t.Error("should skip rules with NameID < 0")
	}
}

func TestEmitViewSkipsLowercaseRules(t *testing.T) {
	rules := map[string]RuleInfo{
		"arrayClose": {Name: "arrayClose", Kind: RuleLeaf, NameID: 5},
		"eof":        {Name: "eof", Kind: RuleLeaf, NameID: 6},
		"Value":      {Name: "Value", Kind: RuleLeaf, NameID: 0},
	}

	code := emitViewTypes(rules, "")
	if strings.Contains(code, "arrayClose") {
		t.Error("should skip lowercase rules")
	}
	if strings.Contains(code, "eof") {
		t.Error("should skip lowercase rules")
	}
	if !strings.Contains(code, "Value") {
		t.Error("should include uppercase rules")
	}
}

func TestEmitViewRootFirst(t *testing.T) {
	rules := map[string]RuleInfo{
		"Zebra": {Name: "Zebra", Kind: RuleLeaf, NameID: 0},
		"Alpha": {Name: "Alpha", Kind: RuleLeaf, NameID: 1},
		"Root":  {Name: "Root", Kind: RuleLeaf, NameID: 2},
	}

	code := emitViewTypes(rules, "Root")
	// Root is reachable → exported "Root".
	// Alpha and Zebra are unreachable → unexported "alpha_view", "zebra_view".
	rootIdx := strings.Index(code, "type Root struct")
	alphaIdx := strings.Index(code, "type alpha_view struct")
	zebraIdx := strings.Index(code, "type zebra_view struct")

	if rootIdx < 0 || alphaIdx < 0 || zebraIdx < 0 {
		t.Fatalf("missing view types in:\n%s", code)
	}
	if rootIdx > alphaIdx {
		t.Error("root should appear before alpha")
	}
	if alphaIdx > zebraIdx {
		t.Error("alpha should appear before zebra (alphabetical)")
	}
}

func TestRenderViewsFile(t *testing.T) {
	rules := map[string]RuleInfo{
		"Root": {Name: "Root", Kind: RuleLeaf, NameID: 0},
		"Item": {Name: "Item", Kind: RuleLeaf, NameID: 1},
	}

	output, err := RenderViewsFile("mypkg", "test.peg", rules, "Root")
	if err != nil {
		t.Fatal(err)
	}

	checks := []string{
		"package mypkg",
		"DO NOT EDIT",
		"_nameID_Root",
		"_nameID_Item",
		"type Root struct",
		"type item_view struct",
	}
	for _, c := range checks {
		if !strings.Contains(output, c) {
			t.Errorf("output missing %q", c)
		}
	}
}
