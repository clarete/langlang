package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationJSON(t *testing.T) {
	dir := t.TempDir()
	grammarPath := jsonGrammarPath()

	src := filepath.Join(dir, "json_types.go")
	err := os.WriteFile(src, []byte(`package json

type JSONValue struct {
	Object *JSONObject `+"`"+`ll:"Object"`+"`"+`
	Array  *JSONArray  `+"`"+`ll:"Array"`+"`"+`
	String *string     `+"`"+`ll:"String"`+"`"+`
	Number *string     `+"`"+`ll:"Number"`+"`"+`
}

type JSONObject struct {
	Members []JSONMember `+"`"+`ll:"Member"`+"`"+`
}

type JSONMember struct {
	Key   string    `+"`"+`ll:"String"`+"`"+`
	Value JSONValue `+"`"+`ll:"Value"`+"`"+`
}

type JSONArray struct {
	Items []JSONValue `+"`"+`ll:"Value"`+"`"+`
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = Generate(src, grammarPath)
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "json_types_extract.go")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	output := string(data)

	checks := []string{
		"DO NOT EDIT",
		"package json",
		// nameID constants
		"_nameID_Object",
		"_nameID_Array",
		"_nameID_String",
		"_nameID_Number",
		"_nameID_Member",
		"_nameID_Value",
		// arena-direct access patterns
		"*tree",
		"t.IsNamed(",
		"t.NameID(cid)",
		// extraction functions
		"ExtractJSONValue",
		"ExtractJSONObject",
		"ExtractJSONMember",
		"ExtractJSONArray",
		// tree operations
		"NodeType_Node",
		"NodeType_String",
		"t.Text(",
		"t.Child(",
		"t.Visit(",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q", check)
		}
	}

	t.Logf("Generated output:\n%s", output)
}

func TestIntegrationJSONViews(t *testing.T) {
	dir := t.TempDir()
	grammarPath := jsonGrammarPath()

	err := GenerateViews(grammarPath, "jsonviews", dir, "JSON")
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "json_views.go")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	output := string(data)

	checks := []string{
		"DO NOT EDIT",
		"package jsonviews",
		"-mode=views",
		// nameID constants for rules
		"_nameID_JSON",
		"_nameID_Value",
		"_nameID_Object",
		"_nameID_Array",
		"_nameID_String",
		"_nameID_Number",
		"_nameID_Member",
		// view types
		"type JSON struct",
		"type Value struct",
		"type Object struct",
		"type Member struct",
		"type Array struct",
		"type String struct",
		// accessor patterns
		"func (v Value) Object() (Object, bool)",
		"func (v Value) StringNode() (String, bool)",
		"func (v Value) Number() (Number, bool)",
		// literal choice accessors
		"func (v Value) IsTrue() bool",
		"func (v Value) IsFalse() bool",
		"func (v Value) IsNull() bool",
		// tree operations
		"*tree",
		"NodeID",
		"Text",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q", check)
		}
	}

	t.Logf("Generated views output:\n%s", output)
}
