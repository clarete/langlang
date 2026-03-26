package tomlextract

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
	base := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "toml")

	inputs := make(map[string][]byte, len(inputNames))
	for _, name := range inputNames {
		data, err := os.ReadFile(filepath.Join(base, "input_"+name+".toml"))
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
	p := NewTOMLParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(name, func(b *testing.B) {
			input := inputs[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				_, err := p.ParseTOML()
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
	p := NewTOMLParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(name, func(b *testing.B) {
			input := inputs[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				parsed, err := p.ParseTOML()
				if err != nil {
					b.Fatal(err)
				}
				tr := parsed.(*tree)
				root, ok := parsed.Root()
				if !ok {
					b.Fatal("no root")
				}
				_, err = ExtractTOMLDoc(tr, root)
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
	p := NewTOMLParser()
	p.SetShowFails(false)

	for _, name := range inputNames {
		b.Run(name, func(b *testing.B) {
			input := inputs[name]
			b.SetBytes(int64(len(input)))
			p.SetInput(input)

			for n := 0; n < b.N; n++ {
				parsed, err := p.ParseTOML()
				if err != nil {
					b.Fatal(err)
				}
				root, ok := parsed.Root()
				if !ok {
					b.Fatal("no root")
				}
				_, err = extractTOMLDocInterface(parsed, root)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Hand-written Tree interface versions for benchmark comparison.

func extractTOMLDocInterface(t Tree, id NodeID) (TOMLDoc, error) {
	var out TOMLDoc
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		if t.Name(cid) == "Expression" {
			val, err := extractTOMLExpressionInterface(t, cid)
			if err == nil {
				out.Expressions = append(out.Expressions, val)
			}
			return false
		}
		return true
	})
	return out, nil
}

func extractTOMLExpressionInterface(t Tree, id NodeID) (TOMLExpression, error) {
	var out TOMLExpression
	child, ok := t.Child(id)
	if !ok {
		return out, fmt.Errorf("TOMLExpression: no child")
	}
	if t.Type(child) != NodeType_Node {
		return out, nil
	}
	switch t.Name(child) {
	case "Table":
		val, err := extractTOMLTableInterface(t, child)
		if err != nil {
			return out, err
		}
		out.Table = &val
	case "KeyVal":
		val, err := extractTOMLKeyValInterface(t, child)
		if err != nil {
			return out, err
		}
		out.KeyVal = &val
	}
	return out, nil
}

func extractTOMLTableInterface(t Tree, id NodeID) (TOMLTable, error) {
	var out TOMLTable
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		switch t.Name(cid) {
		case "Key":
			val, _ := extractTOMLKeyInterface(t, cid)
			out.Key = val
			return false
		case "KeyVal":
			val, err := extractTOMLKeyValInterface(t, cid)
			if err == nil {
				out.KeyVals = append(out.KeyVals, val)
			}
			return false
		}
		return true
	})
	return out, nil
}

func extractTOMLKeyValInterface(t Tree, id NodeID) (TOMLKeyVal, error) {
	var out TOMLKeyVal
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		switch t.Name(cid) {
		case "Key":
			val, _ := extractTOMLKeyInterface(t, cid)
			out.Key = val
			return false
		case "Val":
			val, _ := extractTOMLValInterface(t, cid)
			out.Val = val
			return false
		}
		return true
	})
	return out, nil
}

func extractTOMLKeyInterface(t Tree, id NodeID) (TOMLKey, error) {
	var out TOMLKey
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		if t.Name(cid) == "SimpleKey" {
			val, _ := extractTOMLSimpleKeyInterface(t, cid)
			out.SimpleKeys = append(out.SimpleKeys, val)
			return false
		}
		return true
	})
	return out, nil
}

func extractTOMLSimpleKeyInterface(t Tree, id NodeID) (TOMLSimpleKey, error) {
	var out TOMLSimpleKey
	child, ok := t.Child(id)
	if !ok {
		return out, fmt.Errorf("TOMLSimpleKey: no child")
	}
	if t.Type(child) != NodeType_Node {
		return out, nil
	}
	switch t.Name(child) {
	case "BareKey":
		s := t.Text(child)
		out.BareKey = &s
	case "BasicString":
		s := t.Text(child)
		out.QuotedKey = &s
	}
	return out, nil
}

func extractTOMLValInterface(t Tree, id NodeID) (TOMLVal, error) {
	var out TOMLVal
	child, ok := t.Child(id)
	if !ok {
		return out, fmt.Errorf("TOMLVal: no child")
	}
	if t.Type(child) != NodeType_Node {
		return out, nil
	}
	switch t.Name(child) {
	case "InlineTable":
		val, err := extractTOMLInlineTableInterface(t, child)
		if err != nil {
			return out, err
		}
		out.InlineTable = &val
	case "Array":
		val, err := extractTOMLArrayInterface(t, child)
		if err != nil {
			return out, err
		}
		out.Array = &val
	case "BasicString":
		s := t.Text(child)
		out.String = &s
	case "Number":
		s := t.Text(child)
		out.Number = &s
	case "Boolean":
		s := t.Text(child)
		out.Boolean = &s
	case "DateTime":
		s := t.Text(child)
		out.DateTime = &s
	}
	return out, nil
}

func extractTOMLArrayInterface(t Tree, id NodeID) (TOMLArray, error) {
	var out TOMLArray
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		if t.Name(cid) == "Val" {
			val, _ := extractTOMLValInterface(t, cid)
			out.Items = append(out.Items, val)
			return false
		}
		return true
	})
	return out, nil
}

func extractTOMLInlineTableInterface(t Tree, id NodeID) (TOMLInlineTable, error) {
	var out TOMLInlineTable
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		if t.Name(cid) == "InlineKeyVal" {
			val, _ := extractTOMLInlineKeyValInterface(t, cid)
			out.KeyVals = append(out.KeyVals, val)
			return false
		}
		return true
	})
	return out, nil
}

func extractTOMLInlineKeyValInterface(t Tree, id NodeID) (TOMLInlineKeyVal, error) {
	var out TOMLInlineKeyVal
	t.Visit(id, func(cid NodeID) bool {
		if cid == id {
			return true
		}
		if t.Type(cid) != NodeType_Node {
			return true
		}
		switch t.Name(cid) {
		case "Key":
			val, _ := extractTOMLKeyInterface(t, cid)
			out.Key = val
			return false
		case "Val":
			val, _ := extractTOMLValInterface(t, cid)
			out.Val = val
			return false
		}
		return true
	})
	return out, nil
}
