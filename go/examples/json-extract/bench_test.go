package jsonextract

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var inputNames = []string{"30kb", "500kb"}

func benchInputs(b *testing.B) map[string][]byte {
	b.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "json")

	inputs := make(map[string][]byte, len(inputNames))
	for _, name := range inputNames {
		data, err := os.ReadFile(filepath.Join(base, "input_"+name+".json"))
		if err != nil {
			b.Fatalf("read %s: %v", name, err)
		}
		inputs[name] = data
	}
	return inputs
}

// BenchmarkParseOnly measures raw parsing without extraction.
func BenchmarkParseOnly(b *testing.B) {
	inputs := benchInputs(b)
	p := NewJSONParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(name, func(b *testing.B) {
			input := inputs[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				_, err := p.ParseJSON()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParseAndExtract measures parsing + arena-direct extraction.
func BenchmarkParseAndExtract(b *testing.B) {
	inputs := benchInputs(b)
	p := NewJSONParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(name, func(b *testing.B) {
			input := inputs[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				parsed, err := p.ParseJSON()
				if err != nil {
					b.Fatal(err)
				}
				tr := parsed.(*tree)
				root, ok := parsed.Root()
				if !ok {
					b.Fatal("no root")
				}
				// Find Value node
				var valueID NodeID
				tr.Visit(root, func(id NodeID) bool {
					if id == root {
						return true
					}
					if tr.IsNamed(id, _nameID_Value) {
						valueID = id
						return false
					}
					return true
				})
				_, err = ExtractJSONValue(tr, valueID)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParseAndExtractInterface measures parsing + Tree interface
// extraction (string-based name matching) for comparison.
func BenchmarkParseAndExtractInterface(b *testing.B) {
	inputs := benchInputs(b)
	p := NewJSONParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(name, func(b *testing.B) {
			input := inputs[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				parsed, err := p.ParseJSON()
				if err != nil {
					b.Fatal(err)
				}
				root, ok := parsed.Root()
				if !ok {
					b.Fatal("no root")
				}
				var valueID NodeID
				parsed.Visit(root, func(id NodeID) bool {
					if id == root {
						return true
					}
					if parsed.Type(id) == NodeType_Node && parsed.Name(id) == "Value" {
						valueID = id
						return false
					}
					return true
				})
				_, err = extractJSONValueInterface(parsed, valueID)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// extractJSONValueInterface is a hand-written Tree interface version
// for benchmark comparison against the arena-direct generated code.
func extractJSONValueInterface(t Tree, id NodeID) (JSONValue, error) {
	var out JSONValue
	child, ok := t.Child(id)
	if !ok {
		return out, fmt.Errorf("JSONValue: no child")
	}

	childType := t.Type(child)
	if childType != NodeType_Node {
		// literal alternative (e.g., 'true', 'false', 'null')
		return out, nil
	}
	childName := t.Name(child)

	switch {
	case childName == "Object":
		val, err := extractJSONObjectInterface(t, child)
		if err != nil {
			return out, err
		}
		out.Object = &val
	case childName == "Array":
		val, err := extractJSONArrayInterface(t, child)
		if err != nil {
			return out, err
		}
		out.Array = &val
	case childName == "String":
		s := t.Text(child)
		out.String = &s
	case childName == "Number":
		s := t.Text(child)
		out.Number = &s
	}
	return out, nil
}

func extractJSONObjectInterface(t Tree, id NodeID) (JSONObject, error) {
	var out JSONObject
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		if t.Name(cid) == "Member" {
			val, err := extractJSONMemberInterface(t, cid)
			if err == nil {
				out.Members = append(out.Members, val)
			}
			return false
		}
		return true
	})
	return out, nil
}

func extractJSONMemberInterface(t Tree, id NodeID) (JSONMember, error) {
	var out JSONMember
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		switch t.Name(cid) {
		case "String":
			out.Key = t.Text(cid)
			return false
		case "Value":
			val, err := extractJSONValueInterface(t, cid)
			if err == nil {
				out.Value = val
			}
			return false
		}
		return true
	})
	return out, nil
}

func extractJSONArrayInterface(t Tree, id NodeID) (JSONArray, error) {
	var out JSONArray
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		if t.Name(cid) == "Value" {
			val, err := extractJSONValueInterface(t, cid)
			if err == nil {
				out.Items = append(out.Items, val)
			}
			return false
		}
		return true
	})
	return out, nil
}
