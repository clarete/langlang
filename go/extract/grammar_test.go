package extract

import (
	"path/filepath"
	"runtime"
	"testing"
)

func jsonGrammarPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..",
		"docs", "live", "assets", "examples", "json", "json.peg")
}

func TestAnalyzeGrammar(t *testing.T) {
	rules, err := AnalyzeGrammar(jsonGrammarPath())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		kind RuleKind
	}{
		{"Value", RuleChoice},
		{"Object", RuleSequence},
		{"Array", RuleSequence},
		{"Member", RuleSequence},
		{"String", RuleSequence},
		{"Number", RuleSequence},
		{"Hex", RuleLeaf},
	}
	for _, tt := range tests {
		ri, ok := rules[tt.name]
		if !ok {
			t.Errorf("rule %q not found", tt.name)
			continue
		}
		if ri.Kind != tt.kind {
			t.Errorf("rule %q: got kind %d, want %d", tt.name, ri.Kind, tt.kind)
		}
		if ri.NameID < 0 {
			t.Errorf("rule %q: nameID not resolved", tt.name)
		}
	}

	// Value should have choices including Object, Array, String, Number
	val := rules["Value"]
	if len(val.Choices) < 4 {
		t.Errorf("Value: expected at least 4 choices, got %d: %v",
			len(val.Choices), val.Choices)
	}

	// Member should have children: String (named), ':' (literal), Value (named)
	mem := rules["Member"]
	if len(mem.Children) < 2 {
		t.Errorf("Member: expected at least 2 children, got %d", len(mem.Children))
	}

	// Array <- '[' (Value (',' Value)*)? ']' — Value must appear as a
	// named child so views/extract can generate accessors for it.
	arr := rules["Array"]
	hasValue := false
	for _, c := range arr.Children {
		if c.RuleName == "Value" {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t.Errorf("Array: Value not found in children; got %v", arr.Children)
	}

	// Object <- '{' (Member (',' Member)*)? '}' — same for Member.
	obj := rules["Object"]
	hasMember := false
	for _, c := range obj.Children {
		if c.RuleName == "Member" {
			hasMember = true
			break
		}
	}
	if !hasMember {
		t.Errorf("Object: Member not found in children; got %v", obj.Children)
	}
}
