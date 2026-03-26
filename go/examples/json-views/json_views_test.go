package jsonviews

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func parseJSON(t *testing.T, input string) JSON {
	t.Helper()
	p := NewJSONParser()
	p.SetInput([]byte(input))
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	root, ok := parsed.Root()
	if !ok {
		t.Fatalf("no root for %q", input)
	}
	return newJSON(parsed.(*tree), root)
}

func TestViewString(t *testing.T) {
	json := parseJSON(t, `"hello"`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	str, ok := val.StringNode()
	if !ok {
		t.Fatal("expected String alternative")
	}
	if str.String() != `"hello"` {
		t.Errorf("String.String() = %q, want %q", str.String(), `"hello"`)
	}
}

func TestViewNumber(t *testing.T) {
	json := parseJSON(t, `42`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	num, ok := val.Number()
	if !ok {
		t.Fatal("expected Number alternative")
	}
	if num.String() != "42" {
		t.Errorf("Number.String() = %q, want %q", num.String(), "42")
	}
}

func TestViewObject(t *testing.T) {
	json := parseJSON(t, `{"name": "test"}`)

	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	obj, ok := val.Object()
	if !ok {
		t.Fatal("expected Object alternative")
	}

	if obj.MemberCount() == 0 {
		t.Fatal("no Member in Object")
	}
	mem := obj.MemberAt(0)
	key, ok := mem.StringNode()
	if !ok {
		t.Fatal("no String in Member")
	}
	if key.String() != `"name"` {
		t.Errorf("Member key = %q, want %q", key.String(), `"name"`)
	}

	mval, ok := mem.Value()
	if !ok {
		t.Fatal("no Value in Member")
	}
	str, ok := mval.StringNode()
	if !ok {
		t.Fatal("expected String value")
	}
	if str.String() != `"test"` {
		t.Errorf("Member value = %q, want %q", str.String(), `"test"`)
	}
}

func TestViewArrayAllValues(t *testing.T) {
	json := parseJSON(t, `[1, 2, 3]`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	arr, ok := val.Array()
	if !ok {
		t.Fatal("expected Array alternative")
	}

	// Array should expose all values, not just the first.
	if arr.ValueCount() != 3 {
		t.Fatalf("ValueCount() = %d, want 3", arr.ValueCount())
	}

	want := []string{"1", "2", "3"}
	for i, w := range want {
		v := arr.ValueAt(i)
		num, ok := v.Number()
		if !ok {
			t.Fatalf("item %d: expected Number", i)
		}
		if num.String() != w {
			t.Errorf("item %d = %q, want %q", i, num.String(), w)
		}
	}
}

func TestViewObjectAllMembers(t *testing.T) {
	json := parseJSON(t, `{"a": 1, "b": 2}`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	obj, ok := val.Object()
	if !ok {
		t.Fatal("expected Object alternative")
	}

	// Object should expose all members, not just the first.
	if obj.MemberCount() != 2 {
		t.Fatalf("MemberCount() = %d, want 2", obj.MemberCount())
	}

	wantKeys := []string{`"a"`, `"b"`}
	for i, wk := range wantKeys {
		mem := obj.MemberAt(i)
		key, ok := mem.StringNode()
		if !ok {
			t.Fatalf("member %d: no String key", i)
		}
		if key.String() != wk {
			t.Errorf("member %d key = %q, want %q", i, key.String(), wk)
		}
	}
}

func TestViewPublicConstructor(t *testing.T) {
	p := NewJSONParser()
	p.SetInput([]byte(`42`))
	parsed, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	// Users should be able to create a root view without casting to *tree.
	json := NewJSON(parsed)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	num, ok := val.Number()
	if !ok {
		t.Fatal("expected Number")
	}
	if num.String() != "42" {
		t.Errorf("got %q, want %q", num.String(), "42")
	}
}

func TestViewMemberFmtStringer(t *testing.T) {
	json := parseJSON(t, `{"key": "val"}`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value")
	}
	obj, ok := val.Object()
	if !ok {
		t.Fatal("no Object")
	}
	mem := obj.MemberAt(0)

	// Member should be usable with fmt.Sprintf("%s") without compile error.
	// This tests that Member satisfies fmt.Stringer (has String() string).
	s := fmt.Sprintf("%s", mem)
	if s != `"key": "val"` {
		t.Errorf("fmt.Sprintf(\"%%s\", member) = %q, want %q", s, `"key": "val"`)
	}
}

func TestViewLiteralTrue(t *testing.T) {
	json := parseJSON(t, `true`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	if !val.IsTrue() {
		t.Error("expected IsTrue()")
	}
	if val.IsFalse() {
		t.Error("unexpected IsFalse()")
	}
	if val.IsNull() {
		t.Error("unexpected IsNull()")
	}
}

func TestViewLiteralFalse(t *testing.T) {
	json := parseJSON(t, `false`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	if !val.IsFalse() {
		t.Error("expected IsFalse()")
	}
	if val.IsTrue() {
		t.Error("unexpected IsTrue()")
	}
}

func TestViewLiteralNull(t *testing.T) {
	json := parseJSON(t, `null`)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no Value child")
	}
	if !val.IsNull() {
		t.Error("expected IsNull()")
	}
	if val.IsTrue() {
		t.Error("unexpected IsTrue()")
	}
}

// valueCounts tracks how many of each Value alternative were seen.
type valueCounts struct {
	objects  int
	arrays   int
	strings  int
	numbers  int
	trues    int
	falses   int
	nulls    int
	members  int
	unknown  int
}

func countValue(v Value, c *valueCounts) {
	if obj, ok := v.Object(); ok {
		c.objects++
		for i := 0; i < obj.MemberCount(); i++ {
			c.members++
			mem := obj.MemberAt(i)
			if _, ok := mem.StringNode(); !ok {
				c.unknown++
			}
			if val, ok := mem.Value(); ok {
				countValue(val, c)
			} else {
				c.unknown++
			}
		}
	} else if arr, ok := v.Array(); ok {
		c.arrays++
		for i := 0; i < arr.ValueCount(); i++ {
			countValue(arr.ValueAt(i), c)
		}
	} else if _, ok := v.StringNode(); ok {
		c.strings++
	} else if _, ok := v.Number(); ok {
		c.numbers++
	} else if v.IsTrue() {
		c.trues++
	} else if v.IsFalse() {
		c.falses++
	} else if v.IsNull() {
		c.nulls++
	} else {
		c.unknown++
	}
}

func TestViewLargeDocument(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(thisFile), "..", "..", "..",
		"testdata", "json")

	for _, name := range []string{"30kb", "500kb", "2000kb"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, "input_"+name+".json"))
			if err != nil {
				t.Skipf("test data not available: %v", err)
			}

			p := NewJSONParser()
			p.SetShowFails(false)
			p.SetInput(data)
			parsed, err := p.ParseJSON()
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			json := NewJSON(parsed)
			val, ok := json.Value()
			if !ok {
				t.Fatal("no root Value")
			}

			var c valueCounts
			countValue(val, &c)

			t.Logf("objects=%d arrays=%d strings=%d numbers=%d true=%d false=%d null=%d members=%d",
				c.objects, c.arrays, c.strings, c.numbers, c.trues, c.falses, c.nulls, c.members)

			if c.unknown > 0 {
				t.Errorf("found %d values that matched no alternative", c.unknown)
			}

			total := c.objects + c.arrays + c.strings + c.numbers + c.trues + c.falses + c.nulls
			if total == 0 {
				t.Error("walked zero values")
			}

			// Every object must have at least one member with a string key and a value.
			if c.objects > 0 && c.members == 0 {
				t.Error("objects found but no members")
			}
			if c.members > 0 && c.strings == 0 {
				t.Error("members found but no string values anywhere")
			}
		})
	}
}

func TestViewDeeplyNested(t *testing.T) {
	// Build a deeply nested JSON document programmatically.
	// Structure: {"a": {"a": {"a": ... {"a": [1, true, null, "x"]} ...}}}
	depth := 200
	var b []byte
	for i := 0; i < depth; i++ {
		b = append(b, `{"a": `...)
	}
	b = append(b, `[1, true, null, "x", {"b": false}]`...)
	for i := 0; i < depth; i++ {
		b = append(b, '}')
	}

	p := NewJSONParser()
	p.SetShowFails(false)
	p.SetInput(b)
	parsed, err := p.ParseJSON()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	json := NewJSON(parsed)
	val, ok := json.Value()
	if !ok {
		t.Fatal("no root Value")
	}

	// Walk down the nested objects.
	for i := 0; i < depth; i++ {
		obj, ok := val.Object()
		if !ok {
			t.Fatalf("depth %d: expected Object", i)
		}
		if obj.MemberCount() != 1 {
			t.Fatalf("depth %d: MemberCount() = %d, want 1", i, obj.MemberCount())
		}
		mem := obj.MemberAt(0)
		key, ok := mem.StringNode()
		if !ok {
			t.Fatalf("depth %d: no key", i)
		}
		if key.String() != `"a"` {
			t.Fatalf("depth %d: key = %q, want %q", i, key.String(), `"a"`)
		}
		val, ok = mem.Value()
		if !ok {
			t.Fatalf("depth %d: no value", i)
		}
	}

	// At the bottom: [1, true, null, "x", {"b": false}]
	arr, ok := val.Array()
	if !ok {
		t.Fatal("leaf: expected Array")
	}
	if arr.ValueCount() != 5 {
		t.Fatalf("leaf array: ValueCount() = %d, want 5", arr.ValueCount())
	}

	// 1
	if num, ok := arr.ValueAt(0).Number(); !ok {
		t.Error("item 0: expected Number")
	} else if num.String() != "1" {
		t.Errorf("item 0 = %q, want %q", num.String(), "1")
	}

	// true
	if !arr.ValueAt(1).IsTrue() {
		t.Error("item 1: expected IsTrue()")
	}

	// null
	if !arr.ValueAt(2).IsNull() {
		t.Error("item 2: expected IsNull()")
	}

	// "x"
	if str, ok := arr.ValueAt(3).StringNode(); !ok {
		t.Error("item 3: expected String")
	} else if str.String() != `"x"` {
		t.Errorf("item 3 = %q, want %q", str.String(), `"x"`)
	}

	// {"b": false}
	innerObj, ok := arr.ValueAt(4).Object()
	if !ok {
		t.Fatal("item 4: expected Object")
	}
	if innerObj.MemberCount() != 1 {
		t.Fatalf("item 4: MemberCount() = %d, want 1", innerObj.MemberCount())
	}
	innerMem := innerObj.MemberAt(0)
	innerKey, ok := innerMem.StringNode()
	if !ok {
		t.Fatal("item 4: no key")
	}
	if innerKey.String() != `"b"` {
		t.Errorf("item 4 key = %q, want %q", innerKey.String(), `"b"`)
	}
	innerVal, ok := innerMem.Value()
	if !ok {
		t.Fatal("item 4: no value")
	}
	if !innerVal.IsFalse() {
		t.Error("item 4 value: expected IsFalse()")
	}
}

func TestViewText(t *testing.T) {
	json := parseJSON(t, `{"a": 1}`)
	if json.String() != `{"a": 1}` {
		t.Errorf("JSON.String() = %q, want %q", json.String(), `{"a": 1}`)
	}
}
