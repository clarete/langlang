package extract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyze(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "types.go")
	err := os.WriteFile(src, []byte(`package example

type JSONValue struct {
	Object *JSONObject `+"`"+`ll:"Object"`+"`"+`
	String *string     `+"`"+`ll:"String"`+"`"+`
	Raw    *string
}

type JSONObject struct {
	Members []JSONMember `+"`"+`ll:"Member"`+"`"+`
}

type JSONMember struct {
	Key   string    `+"`"+`ll:"String"`+"`"+`
	Value JSONValue `+"`"+`ll:"Value"`+"`"+`
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	structs, err := Analyze(src)
	if err != nil {
		t.Fatal(err)
	}

	if len(structs) != 3 {
		t.Fatalf("expected 3 structs, got %d", len(structs))
	}

	// JSONValue: 2 tagged fields (Object, String); Raw has no ll tag
	jv := findStruct(structs, "JSONValue")
	if jv == nil {
		t.Fatal("JSONValue not found")
	}
	if len(jv.Fields) != 2 {
		t.Errorf("JSONValue: expected 2 tagged fields, got %d", len(jv.Fields))
	}

	// JSONObject: 1 tagged field (Members)
	jo := findStruct(structs, "JSONObject")
	if jo == nil {
		t.Fatal("JSONObject not found")
	}
	if len(jo.Fields) != 1 {
		t.Errorf("JSONObject: expected 1 tagged field, got %d", len(jo.Fields))
	}
	if jo.Fields[0].Kind != FieldSlice {
		t.Errorf("JSONObject.Members: expected FieldSlice, got %d", jo.Fields[0].Kind)
	}

	// JSONMember: Key is string -> FieldText, Value is struct -> FieldNamedRule
	jm := findStruct(structs, "JSONMember")
	if jm == nil {
		t.Fatal("JSONMember not found")
	}
	if jm.Fields[0].Kind != FieldText {
		t.Errorf("JSONMember.Key: expected FieldText, got %d", jm.Fields[0].Kind)
	}
	if jm.Fields[1].Kind != FieldNamedRule {
		t.Errorf("JSONMember.Value: expected FieldNamedRule, got %d", jm.Fields[1].Kind)
	}
}

func findStruct(structs []StructInfo, name string) *StructInfo {
	for i := range structs {
		if structs[i].Name == name {
			return &structs[i]
		}
	}
	return nil
}
