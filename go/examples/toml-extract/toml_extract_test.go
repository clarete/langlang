package tomlextract

import (
	"testing"
)

func parseAndExtract(t *testing.T, input string) TOMLDoc {
	t.Helper()
	p := NewTOMLParser()
	p.SetInput([]byte(input))
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", input)
	}
	root, ok := parsed.Root()
	if !ok {
		t.Fatal("no root")
	}
	tr := parsed.(*tree)

	doc, err := ExtractTOMLDoc(tr, root)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return doc
}

// keyName returns the string representation of a TOMLKey.
func keyName(k TOMLKey) string {
	if len(k.SimpleKeys) == 0 {
		return ""
	}
	sk := k.SimpleKeys[0]
	if sk.BareKey != nil {
		return *sk.BareKey
	}
	if sk.QuotedKey != nil {
		return *sk.QuotedKey
	}
	return ""
}

func TestExtractStringKeyVal(t *testing.T) {
	doc := parseAndExtract(t, `title = "hello"`)
	if len(doc.Expressions) != 1 {
		t.Fatalf("expected 1 expression, got %d", len(doc.Expressions))
	}
	expr := doc.Expressions[0]
	if expr.KeyVal == nil {
		t.Fatal("expected KeyVal")
	}
	if keyName(expr.KeyVal.Key) != "title" {
		t.Errorf("key = %q, want %q", keyName(expr.KeyVal.Key), "title")
	}
	if expr.KeyVal.Val.String == nil {
		t.Fatal("expected String value")
	}
	if *expr.KeyVal.Val.String != `"hello"` {
		t.Errorf("val = %q, want %q", *expr.KeyVal.Val.String, `"hello"`)
	}
}

func TestExtractNumber(t *testing.T) {
	doc := parseAndExtract(t, `count = 42`)
	kv := doc.Expressions[0].KeyVal
	if kv.Val.Number == nil {
		t.Fatal("expected Number value")
	}
	if *kv.Val.Number != "42" {
		t.Errorf("val = %q, want %q", *kv.Val.Number, "42")
	}
}

func TestExtractFloat(t *testing.T) {
	doc := parseAndExtract(t, `ratio = 3.14`)
	kv := doc.Expressions[0].KeyVal
	if kv.Val.Number == nil {
		t.Fatal("expected Number value")
	}
	if *kv.Val.Number != "3.14" {
		t.Errorf("val = %q, want %q", *kv.Val.Number, "3.14")
	}
}

func TestExtractBoolean(t *testing.T) {
	doc := parseAndExtract(t, `enabled = true`)
	kv := doc.Expressions[0].KeyVal
	if kv.Val.Boolean == nil {
		t.Fatal("expected Boolean value")
	}
	if *kv.Val.Boolean != "true" {
		t.Errorf("val = %q, want %q", *kv.Val.Boolean, "true")
	}
}

func TestExtractTable(t *testing.T) {
	input := "[server]\nhost = \"localhost\"\nport = 8080\n"
	doc := parseAndExtract(t, input)
	if len(doc.Expressions) != 1 {
		t.Fatalf("expected 1 expression, got %d", len(doc.Expressions))
	}
	tbl := doc.Expressions[0].Table
	if tbl == nil {
		t.Fatal("expected Table")
	}
	if keyName(tbl.Key) != "server" {
		t.Errorf("table key = %q, want %q", keyName(tbl.Key), "server")
	}
	if len(tbl.KeyVals) != 2 {
		t.Fatalf("expected 2 key-vals in table, got %d", len(tbl.KeyVals))
	}
	if keyName(tbl.KeyVals[0].Key) != "host" {
		t.Errorf("kv[0] key = %q, want %q", keyName(tbl.KeyVals[0].Key), "host")
	}
	if tbl.KeyVals[0].Val.String == nil || *tbl.KeyVals[0].Val.String != `"localhost"` {
		t.Errorf("kv[0] val = %v, want %q", tbl.KeyVals[0].Val.String, `"localhost"`)
	}
	if keyName(tbl.KeyVals[1].Key) != "port" {
		t.Errorf("kv[1] key = %q, want %q", keyName(tbl.KeyVals[1].Key), "port")
	}
	if tbl.KeyVals[1].Val.Number == nil || *tbl.KeyVals[1].Val.Number != "8080" {
		t.Errorf("kv[1] val = %v, want %q", tbl.KeyVals[1].Val.Number, "8080")
	}
}

func TestExtractArray(t *testing.T) {
	doc := parseAndExtract(t, `tags = ["web", "api"]`)
	kv := doc.Expressions[0].KeyVal
	if kv.Val.Array == nil {
		t.Fatal("expected Array value")
	}
	if len(kv.Val.Array.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(kv.Val.Array.Items))
	}
	if kv.Val.Array.Items[0].String == nil || *kv.Val.Array.Items[0].String != `"web"` {
		t.Errorf("item[0] = %v, want %q", kv.Val.Array.Items[0].String, `"web"`)
	}
	if kv.Val.Array.Items[1].String == nil || *kv.Val.Array.Items[1].String != `"api"` {
		t.Errorf("item[1] = %v, want %q", kv.Val.Array.Items[1].String, `"api"`)
	}
}

func TestExtractInlineTable(t *testing.T) {
	doc := parseAndExtract(t, `point = {x = 1, y = 2}`)
	kv := doc.Expressions[0].KeyVal
	if kv.Val.InlineTable == nil {
		t.Fatal("expected InlineTable value")
	}
	it := kv.Val.InlineTable
	if len(it.KeyVals) != 2 {
		t.Fatalf("expected 2 inline key-vals, got %d", len(it.KeyVals))
	}
	if keyName(it.KeyVals[0].Key) != "x" {
		t.Errorf("inline kv[0] key = %q, want %q", keyName(it.KeyVals[0].Key), "x")
	}
	if it.KeyVals[0].Val.Number == nil || *it.KeyVals[0].Val.Number != "1" {
		t.Errorf("inline kv[0] val = %v, want %q", it.KeyVals[0].Val.Number, "1")
	}
}

func TestExtractMultiExpression(t *testing.T) {
	input := "title = \"My App\"\nversion = 1\n\n[database]\nhost = \"localhost\"\nport = 5432\n"
	doc := parseAndExtract(t, input)
	if len(doc.Expressions) != 3 {
		t.Fatalf("expected 3 expressions, got %d", len(doc.Expressions))
	}
	// First two are KeyVals
	if doc.Expressions[0].KeyVal == nil {
		t.Error("expr[0]: expected KeyVal")
	}
	if doc.Expressions[1].KeyVal == nil {
		t.Error("expr[1]: expected KeyVal")
	}
	// Third is a Table
	if doc.Expressions[2].Table == nil {
		t.Error("expr[2]: expected Table")
	}
}

func TestExtractDottedKey(t *testing.T) {
	doc := parseAndExtract(t, `server.host = "localhost"`)
	kv := doc.Expressions[0].KeyVal
	if len(kv.Key.SimpleKeys) != 2 {
		t.Fatalf("expected 2 simple keys, got %d", len(kv.Key.SimpleKeys))
	}
	if *kv.Key.SimpleKeys[0].BareKey != "server" {
		t.Errorf("key[0] = %q, want %q", *kv.Key.SimpleKeys[0].BareKey, "server")
	}
	if *kv.Key.SimpleKeys[1].BareKey != "host" {
		t.Errorf("key[1] = %q, want %q", *kv.Key.SimpleKeys[1].BareKey, "host")
	}
}
