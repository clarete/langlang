package jsonviews

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var inputNames = []string{"30kb", "500kb", "2000kb"}

func benchInputs(b *testing.B) map[string][]byte {
	b.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(thisFile), "..", "..", "..",
		"testdata", "json")

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

// BenchmarkParseOnly measures raw parsing without any tree traversal.
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

// walkValue recursively walks a Value, touching every leaf via views.
func walkValue(v Value) int {
	count := 1
	if obj, ok := v.Object(); ok {
		for i := 0; i < obj.MemberCount(); i++ {
			mem := obj.MemberAt(i)
			_ = mem.String()
			if val, ok := mem.Value(); ok {
				count += walkValue(val)
			}
		}
	} else if arr, ok := v.Array(); ok {
		for i := 0; i < arr.ValueCount(); i++ {
			count += walkValue(arr.ValueAt(i))
		}
	} else if str, ok := v.StringNode(); ok {
		_ = str.String()
	} else if num, ok := v.Number(); ok {
		_ = num.String()
	} else {
		_ = v.IsTrue() || v.IsFalse() || v.IsNull()
	}
	return count
}

func buildNestedJSON(depth int) []byte {
	var b []byte
	for i := 0; i < depth; i++ {
		b = append(b, `{"k": `...)
	}
	b = append(b, `[1, true, null, "x"]`...)
	for i := 0; i < depth; i++ {
		b = append(b, '}')
	}
	return b
}

// BenchmarkNestedParseOnly measures parsing deeply nested documents.
func BenchmarkNestedParseOnly(b *testing.B) {
	p := NewJSONParser()
	p.SetShowFails(false)

	for _, depth := range []int{10, 50, 200} {
		input := buildNestedJSON(depth)
		b.Run(fmt.Sprintf("depth%d", depth), func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			p.SetInput(input)
			for n := 0; n < b.N; n++ {
				if _, err := p.ParseJSON(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkNestedParseAndWalkViews measures parsing + walking deeply nested documents.
func BenchmarkNestedParseAndWalkViews(b *testing.B) {
	p := NewJSONParser()
	p.SetShowFails(false)

	for _, depth := range []int{10, 50, 200} {
		input := buildNestedJSON(depth)
		b.Run(fmt.Sprintf("depth%d", depth), func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			p.SetInput(input)
			for n := 0; n < b.N; n++ {
				parsed, err := p.ParseJSON()
				if err != nil {
					b.Fatal(err)
				}
				json := NewJSON(parsed)
				val, ok := json.Value()
				if !ok {
					b.Fatal("no Value")
				}
				walkValue(val)
			}
		})
	}
}

// BenchmarkParseAndWalkViews measures parsing + view-based tree traversal.
func BenchmarkParseAndWalkViews(b *testing.B) {
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
				json := newJSON(tr, root)
				val, ok := json.Value()
				if !ok {
					b.Fatal("no Value")
				}
				walkValue(val)
			}
		})
	}
}
