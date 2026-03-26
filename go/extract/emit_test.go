package extract

import (
	"strings"
	"testing"
)

func TestEmitChoiceFunction(t *testing.T) {
	si := StructInfo{
		Name: "JSONValue",
		Fields: []FieldInfo{
			{GoName: "Object", LLTag: "Object", Kind: FieldOptional, GoType: "*JSONObject", NameID: 1},
			{GoName: "String", LLTag: "String", Kind: FieldOptional, GoType: "*string", NameID: 3},
		},
	}
	rules := map[string]RuleInfo{
		"Value":  {Name: "Value", Kind: RuleChoice, NameID: 0, Choices: []string{"Object", "String"}},
		"Object": {Name: "Object", Kind: RuleSequence, NameID: 1},
		"String": {Name: "String", Kind: RuleLeaf, NameID: 3},
	}

	code := emitExtractFunction(si, rules, true)

	if !strings.Contains(code, "_nameID_Object") {
		t.Error("missing _nameID_Object reference")
	}
	if !strings.Contains(code, "_nameID_String") {
		t.Error("missing _nameID_String reference")
	}
	if !strings.Contains(code, "ExtractJSONValue") {
		t.Error("missing ExtractJSONValue function")
	}
	if !strings.Contains(code, "t.IsNamed(") {
		t.Error("missing t.IsNamed() call for arena-direct access")
	}
	if !strings.Contains(code, "*tree") {
		t.Error("missing *tree parameter type")
	}
}

func TestEmitSequenceFunction(t *testing.T) {
	si := StructInfo{
		Name: "JSONMember",
		Fields: []FieldInfo{
			{GoName: "Key", LLTag: "String", Kind: FieldText, GoType: "string", NameID: 3},
			{GoName: "Value", LLTag: "Value", Kind: FieldNamedRule, GoType: "JSONValue", NameID: 0},
		},
	}
	rules := map[string]RuleInfo{
		"Member": {Name: "Member", Kind: RuleSequence, NameID: 5,
			Children: []RuleChild{
				{RuleName: "String", Index: 0},
				{IsLiteral: true, Index: 1},
				{RuleName: "Value", Index: 2},
			}},
	}

	code := emitExtractFunction(si, rules, false)

	if !strings.Contains(code, "NodeType_Node") {
		t.Error("missing NodeType_Node check")
	}
	if !strings.Contains(code, "t.Text(") {
		t.Error("missing t.Text() call for string field")
	}
	if !strings.Contains(code, "t.NameID(cid)") {
		t.Error("missing t.NameID() call for arena-direct name matching")
	}
	if !strings.Contains(code, "_nameID_String") {
		t.Error("missing _nameID_String constant reference")
	}
}
