package extract

import (
	"strings"
	"testing"
)

func TestRenderFile(t *testing.T) {
	nameIDs := []NameIDEntry{
		{Name: "Value", ID: 0},
		{Name: "Object", ID: 1},
	}
	structs := []StructInfo{{
		Name: "JSONValue",
		Fields: []FieldInfo{
			{GoName: "Object", LLTag: "Object", Kind: FieldOptional,
				GoType: "*JSONObject", NameID: 1},
		},
	}}
	rules := map[string]RuleInfo{
		"Value":  {Kind: RuleChoice, Choices: []string{"Object"}},
		"Object": {Kind: RuleSequence},
	}

	output, err := RenderFile("example", "test.peg", nameIDs, structs, rules)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "package example") {
		t.Error("missing package declaration")
	}
	if !strings.Contains(output, "DO NOT EDIT") {
		t.Error("missing generated code header")
	}
	if !strings.Contains(output, "_nameID_Value") {
		t.Error("missing nameID constant for Value")
	}
	if !strings.Contains(output, "_nameID_Object") {
		t.Error("missing nameID constant for Object")
	}
	if !strings.Contains(output, "ExtractJSONValue") {
		t.Error("missing ExtractJSONValue function")
	}
}
