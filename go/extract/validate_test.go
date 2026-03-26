package extract

import "testing"

func TestValidateUnknownRule(t *testing.T) {
	structs := []StructInfo{{
		Name:   "Foo",
		Fields: []FieldInfo{{GoName: "Bar", LLTag: "Nonexistent", Kind: FieldText}},
	}}
	rules := map[string]RuleInfo{}

	_, errs := Validate(structs, rules)
	if len(errs) == 0 {
		t.Error("expected error for unknown rule")
	}
}

func TestValidateTypeMismatch(t *testing.T) {
	structs := []StructInfo{{
		Name: "Foo",
		Fields: []FieldInfo{{
			GoName: "Bar",
			LLTag:  "SomeRule",
			Kind:   FieldText,
			GoType: "string",
		}},
	}}
	rules := map[string]RuleInfo{
		"SomeRule": {Name: "SomeRule", Kind: RuleChoice},
	}

	_, errs := Validate(structs, rules)
	if len(errs) == 0 {
		t.Error("expected error for string field on choice rule")
	}
}

func TestValidateChoiceReclassification(t *testing.T) {
	structs := []StructInfo{{
		Name: "Value",
		Fields: []FieldInfo{
			{GoName: "Object", LLTag: "Object", Kind: FieldOptional, GoType: "*JSONObject"},
			{GoName: "Array", LLTag: "Array", Kind: FieldOptional, GoType: "*JSONArray"},
		},
	}}
	rules := map[string]RuleInfo{
		"Value":  {Name: "Value", Kind: RuleChoice, Choices: []string{"Object", "Array"}},
		"Object": {Name: "Object", Kind: RuleSequence},
		"Array":  {Name: "Array", Kind: RuleSequence},
	}

	result, errs := Validate(structs, rules)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if result[0].Fields[0].Kind != FieldOptional {
		t.Errorf("expected fields to remain FieldOptional, got %d", result[0].Fields[0].Kind)
	}
}
