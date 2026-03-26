# Langlang Extract Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Emit compile-time extraction functions that walk langlang parse trees
via direct arena access, driven by `//go:generate langlang extract` directives
on Go structs tagged with PEG rule names. JSON grammar proof-of-concept.

**Architecture:** A new `go/extract/` package that analyzes Go structs
(`go/ast`), loads PEG grammars (`QueryAST`/`QueryBytecode`), cross-validates
struct fields against grammar rules, and emits arena-direct extraction code via
`fmt.Fprintf` + `text/template`. A `langlang extract` CLI subcommand wires it
together.

**Tech Stack:** `go/ast`, `go/parser`, `go/token`, `go/format`, `text/template`,
langlang's `QueryAST`/`QueryBytecode`

**Rollback:** Delete generated `_extract.go` files and remove
`//go:generate langlang extract` directives. All existing interfaces remain
functional.

**Design doc:** `docs/plans/2026-03-25-extract-design.md`

--------------------------------------------------------------------------------

## Key Reference Files

Before starting any task, read these files to understand the types you're
working with:

- `go/tree.go` --- `tree`, `node` structs, `NodeType_*` constants, arena layout
- `go/api.go` --- `Tree` interface, `NodeID`, `NodeType`
- `go/grammar_ast.go` --- `GrammarNode`, `DefinitionNode`, `SequenceNode`,
  `ChoiceNode`, `IdentifierNode`, `LiteralNode`, `OptionalNode`,
  `ZeroOrMoreNode`, `OneOrMoreNode`, `LexNode`, `LabeledNode`, `CaptureNode`
- `go/grammar_ast_visitor.go` --- `AstNodeVisitor`, `Inspect()` function
- `go/query_api.go` --- `QueryAST()`, `QueryBytecode()`
- `go/vm.go:8-17` --- `Bytecode` struct (notably `smap map[string]int`)
- `go/vm_encoder.go` --- `Encode()` function
- `go/gen.go` --- existing codegen patterns, `outputWriter`
- `go/cmd/langlang/main.go` --- existing CLI structure

## Important: Tree Structure Facts

These were verified against the codebase during design and are critical for
correct code generation:

1.  **`NodeType_Node`** has exactly one child (accessed via `childID`). It wraps
    a named rule match. `nameID` indexes into the string table.
2.  **`NodeType_Sequence`** has N children via `childRanges[childID]` -\>
    `children[start:end]`.
3.  **`NodeType_String`** is a leaf --- byte range `input[start:end]`.
4.  Literals in sequences (e.g., `':'` in `Member <- String ':' Value`) produce
    `NodeType_String` children in the sequence.
5.  Literals in choices (e.g., `'true'` in `Value <- ... / 'true'`) produce bare
    `NodeType_String` with no `nameID`.
6.  `ChoiceNode` is binary right-associative: `A / B / C` becomes
    `ChoiceNode{A, ChoiceNode{B, C}}`. Must be flattened.
7.  `LexNode` (`#(...)`) and `LabeledNode` (`^label`) don't affect tree
    structure.
8.  `Bytecode.smap` maps rule name strings to integer IDs. These IDs match
    `node.nameID` in the parse tree.

--------------------------------------------------------------------------------

### Task 1: Types and tag extraction

Define the core types (`FieldKind`, `FieldInfo`, `StructInfo`, `RuleKind`,
`RuleInfo`, `RuleChild`) and the `ll:` tag parser.

**Promotion criteria:** N/A (new code)

**Files:** - Create: `go/extract/types.go` - Create: `go/extract/types_test.go`

**Step 1: Write the failing test**

``` go
// go/extract/types_test.go
package extract

import "testing"

func TestExtractLLTag(t *testing.T) {
    tests := []struct {
        raw     string
        wantKey string
        wantOk  bool
    }{
        {raw: `ll:"Object"`, wantKey: "Object", wantOk: true},
        {raw: `ll:"Value"`, wantKey: "Value", wantOk: true},
        {raw: `json:"foo" ll:"Bar"`, wantKey: "Bar", wantOk: true},
        {raw: `json:"foo"`, wantKey: "", wantOk: false},
        {raw: ``, wantKey: "", wantOk: false},
        {raw: `ll:"-"`, wantKey: "", wantOk: false},
    }
    for _, tt := range tests {
        key, ok := extractLLTag(tt.raw)
        if key != tt.wantKey || ok != tt.wantOk {
            t.Errorf("extractLLTag(%q) = (%q, %v), want (%q, %v)",
                tt.raw, key, ok, tt.wantKey, tt.wantOk)
        }
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test ./extract/ -run TestExtractLLTag -v` Expected: FAIL ---
package doesn't exist yet

**Step 3: Write the types and tag extraction**

``` go
// go/extract/types.go
package extract

import (
    "reflect"
    "strings"
)

// FieldKind classifies a struct field's extraction strategy.
type FieldKind int

const (
    FieldText      FieldKind = iota // terminal rule -> string
    FieldNamedRule                  // rule reference -> struct with ll: tags
    FieldOptional                   // grammar ? or pointer choice branch
    FieldSlice                      // grammar * or + -> slice
    FieldChoice                     // grammar / -> struct with pointer fields
)

// FieldInfo describes a single struct field tagged with ll:"RuleName".
type FieldInfo struct {
    GoName   string      // Go field name
    LLTag    string      // rule name from ll:"..." tag
    Kind     FieldKind   // classification
    GoType   string      // Go type name (e.g., "string", "*JSONObject")
    ElemType string      // for slices: element type name
    Inner    *StructInfo // for nested structs with ll: tags
    NameID   int32       // resolved from bytecode smap (set during validation)
}

// StructInfo describes a struct with ll:-tagged fields.
type StructInfo struct {
    Name   string
    Fields []FieldInfo
}

// RuleKind classifies a grammar rule's expression structure.
type RuleKind int

const (
    RuleLeaf     RuleKind = iota // terminal (literal, charset, range, any)
    RuleSequence                 // SequenceNode
    RuleChoice                   // ChoiceNode (possibly nested)
    RuleRepeat                   // ZeroOrMore or OneOrMore
    RuleOptional                 // OptionalNode
    RuleAlias                    // single IdentifierNode (rule reference)
)

// RuleInfo describes a grammar rule's structure for extraction purposes.
type RuleInfo struct {
    Name     string
    Kind     RuleKind
    NameID   int32       // from Bytecode.smap
    Children []RuleChild // for sequences: ordered children with metadata
    Choices  []string    // for choices: rule names of alternatives
    Inner    string      // for alias/optional/repeat: the referenced rule
}

// RuleChild describes one item in a sequence rule.
type RuleChild struct {
    RuleName  string // rule or capture name; empty for literals
    IsLiteral bool   // true for structural punctuation (dead child)
    Index     int    // position in SequenceNode.Items
}

// extractLLTag parses an ll:"RuleName" tag from a raw struct tag string.
// Returns the rule name and true if found, or ("", false) if absent or "-".
func extractLLTag(raw string) (string, bool) {
    tag := reflect.StructTag(strings.Trim(raw, "`"))
    val, ok := tag.Lookup("ll")
    if !ok || val == "" || val == "-" {
        return "", false
    }
    // Strip options after comma (future: ll:"Rule,text")
    if idx := strings.Index(val, ","); idx >= 0 {
        val = val[:idx]
    }
    return val, true
}
```

**Step 4: Run test to verify it passes**

Run: `cd go && go test ./extract/ -run TestExtractLLTag -v` Expected: PASS

**Step 5: Commit**

Message: `go[extract] add types and tag extraction`

--------------------------------------------------------------------------------

### Task 2: Struct analysis

Parse a Go source file with `go/ast`, find structs with `ll:` tags, and return
`[]StructInfo` with initial Go-type-only field classification.

**Promotion criteria:** N/A

**Files:** - Create: `go/extract/analyze.go` - Create:
`go/extract/analyze_test.go`

**Step 1: Write the failing test**

Create a test that writes a temp Go file with tagged structs and calls
`Analyze()`:

``` go
// go/extract/analyze_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test ./extract/ -run TestAnalyze -v` Expected: FAIL ---
`Analyze` not defined

**Step 3: Write the implementation**

``` go
// go/extract/analyze.go
package extract

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "strings"
)

// Analyze parses a Go source file and returns StructInfo for each struct
// that has at least one field with an ll:"..." tag. Fields without ll tags
// are excluded. Classification is Go-type-only at this stage; grammar-aware
// reclassification happens in Validate.
func Analyze(filename string) ([]StructInfo, error) {
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
    if err != nil {
        return nil, fmt.Errorf("parse %s: %w", filename, err)
    }

    // Collect all struct names that have ll: tags (for Inner resolution)
    taggedStructs := map[string]bool{}
    ast.Inspect(file, func(n ast.Node) bool {
        ts, ok := n.(*ast.TypeSpec)
        if !ok {
            return true
        }
        st, ok := ts.Type.(*ast.StructType)
        if !ok {
            return true
        }
        for _, field := range st.Fields.List {
            if field.Tag != nil {
                if _, ok := extractLLTag(field.Tag.Value); ok {
                    taggedStructs[ts.Name.Name] = true
                    break
                }
            }
        }
        return true
    })

    var structs []StructInfo
    ast.Inspect(file, func(n ast.Node) bool {
        ts, ok := n.(*ast.TypeSpec)
        if !ok {
            return true
        }
        st, ok := ts.Type.(*ast.StructType)
        if !ok {
            return true
        }
        if !taggedStructs[ts.Name.Name] {
            return true
        }

        si := StructInfo{Name: ts.Name.Name}
        for _, field := range st.Fields.List {
            if field.Tag == nil {
                continue
            }
            tag, ok := extractLLTag(field.Tag.Value)
            if !ok {
                continue
            }
            if len(field.Names) == 0 {
                continue
            }

            fi := FieldInfo{
                GoName: field.Names[0].Name,
                LLTag:  tag,
                GoType: typeString(field.Type),
            }
            fi.Kind, fi.ElemType = classifyGoType(field.Type, taggedStructs)
            si.Fields = append(si.Fields, fi)
        }

        if len(si.Fields) > 0 {
            structs = append(structs, si)
        }
        return true
    })

    return structs, nil
}

// classifyGoType classifies a field by its Go type expression.
// This is an initial classification; grammar-aware reclassification
// happens during validation.
func classifyGoType(expr ast.Expr, tagged map[string]bool) (FieldKind, string) {
    switch t := expr.(type) {
    case *ast.Ident:
        if t.Name == "string" {
            return FieldText, ""
        }
        if tagged[t.Name] {
            return FieldNamedRule, ""
        }
        return FieldText, "" // fallback: treat as text

    case *ast.StarExpr:
        // *string or *SomeStruct
        return FieldOptional, ""

    case *ast.ArrayType:
        // []SomeType
        elemType := typeString(t.Elt)
        return FieldSlice, elemType

    default:
        return FieldText, ""
    }
}

// typeString returns a string representation of a Go type expression.
func typeString(expr ast.Expr) string {
    switch t := expr.(type) {
    case *ast.Ident:
        return t.Name
    case *ast.StarExpr:
        return "*" + typeString(t.X)
    case *ast.ArrayType:
        return "[]" + typeString(t.Elt)
    case *ast.SelectorExpr:
        return typeString(t.X) + "." + t.Sel.Name
    default:
        return fmt.Sprintf("%T", expr)
    }
}

// isChoiceStruct checks if a struct has all-pointer ll:-tagged fields,
// indicating it represents an ordered choice.
func isChoiceStruct(si *StructInfo) bool {
    if len(si.Fields) == 0 {
        return false
    }
    for _, f := range si.Fields {
        if !strings.HasPrefix(f.GoType, "*") {
            return false
        }
    }
    return true
}
```

**Step 4: Run test to verify it passes**

Run: `cd go && go test ./extract/ -run TestAnalyze -v` Expected: PASS

**Step 5: Commit**

Message: `go[extract] add struct analysis via go/ast`

--------------------------------------------------------------------------------

### Task 3: Grammar analysis

Load a PEG grammar via `QueryAST` + `QueryBytecode` and build a
`map[string]RuleInfo`. This classifies each rule by its expression AST type and
resolves `nameID` from the bytecode string table.

**Promotion criteria:** N/A

**Files:** - Create: `go/extract/grammar.go` - Create:
`go/extract/grammar_test.go`

**Step 1: Write the failing test**

Test against the JSON grammar at `docs/live/assets/examples/json/json.peg`.
Verify rule classification for `Value` (choice), `Member` (sequence), `Number`
(sequence with optional), `String` (sequence with lex), and leaf rules like
`Hex`.

``` go
// go/extract/grammar_test.go
package extract

import (
    "path/filepath"
    "runtime"
    "testing"
)

func jsonGrammarPath() string {
    _, thisFile, _, _ := runtime.Caller(0)
    return filepath.Join(filepath.Dir(thisFile), "..", "..",
        "docs", "live", "assets", "examples", "json", "json.peg")
}

func TestAnalyzeGrammar(t *testing.T) {
    rules, err := AnalyzeGrammar(jsonGrammarPath())
    if err != nil {
        t.Fatal(err)
    }

    tests := []struct {
        name string
        kind RuleKind
    }{
        {"Value", RuleChoice},
        {"Object", RuleSequence},
        {"Array", RuleSequence},
        {"Member", RuleSequence},
        {"String", RuleSequence},
        {"Number", RuleSequence},
        {"Hex", RuleLeaf},
    }
    for _, tt := range tests {
        ri, ok := rules[tt.name]
        if !ok {
            t.Errorf("rule %q not found", tt.name)
            continue
        }
        if ri.Kind != tt.kind {
            t.Errorf("rule %q: got kind %d, want %d", tt.name, ri.Kind, tt.kind)
        }
        if ri.NameID < 0 {
            t.Errorf("rule %q: nameID not resolved", tt.name)
        }
    }

    // Value should have choices including Object, Array, String, Number
    val := rules["Value"]
    if len(val.Choices) < 4 {
        t.Errorf("Value: expected at least 4 choices, got %d: %v",
            len(val.Choices), val.Choices)
    }

    // Member should have children: String (named), ':' (literal), Value (named)
    mem := rules["Member"]
    if len(mem.Children) < 2 {
        t.Errorf("Member: expected at least 2 children, got %d", len(mem.Children))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test ./extract/ -run TestAnalyzeGrammar -v` Expected: FAIL ---
`AnalyzeGrammar` not defined

**Step 3: Write the implementation**

Key considerations: - Use `langlang.NewDatabase`, `langlang.QueryAST`,
`langlang.QueryBytecode` - Walk `GrammarNode.Definitions`, classify each
`DefinitionNode.Expr` - Flatten binary `ChoiceNode` into a list of
alternatives - For sequences, tag each item as literal vs rule-reference -
Unwrap `LabeledNode`, `LexNode`, `CaptureNode` (they don't affect tree
structure) - Resolve `nameID` via `Bytecode.smap`

``` go
// go/extract/grammar.go
package extract

import (
    "fmt"

    langlang "github.com/clarete/langlang/go"
)

// AnalyzeGrammar loads a PEG grammar and returns a map of rule names to
// RuleInfo, describing each rule's structure for extraction codegen.
func AnalyzeGrammar(grammarPath string) (map[string]RuleInfo, error) {
    cfg := langlang.NewConfig()
    loader := langlang.NewRelativeImportLoader()
    db := langlang.NewDatabase(cfg, loader)

    grammar, err := langlang.QueryAST(db, grammarPath)
    if err != nil {
        return nil, fmt.Errorf("query AST: %w", err)
    }

    bytecode, err := langlang.QueryBytecode(db, grammarPath)
    if err != nil {
        return nil, fmt.Errorf("query bytecode: %w", err)
    }

    rules := make(map[string]RuleInfo, len(grammar.Definitions))
    for _, def := range grammar.Definitions {
        ri := classifyRule(def)
        ri.NameID = resolveNameID(bytecode, def.Name)
        rules[def.Name] = ri
    }

    return rules, nil
}

// resolveNameID looks up a rule name in the bytecode string table.
// Returns -1 if not found.
func resolveNameID(bc *langlang.Bytecode, name string) int32 {
    // Bytecode.smap is unexported. We need to access it.
    // Since we're in a different package, we'll use the Program's
    // StringID method or find another way.
    //
    // TODO: This requires Bytecode.smap to be accessible. Options:
    // 1. Add a public method to Bytecode: func (b *Bytecode) NameID(name string) (int32, bool)
    // 2. Use the Program instead of Bytecode
    // 3. Add the method to the langlang package
    //
    // For now, use QueryProgram which has a public StringID method.
    return -1 // placeholder
}

func classifyRule(def *langlang.DefinitionNode) RuleInfo {
    ri := RuleInfo{Name: def.Name}
    expr := unwrapTransparent(def.Expr)

    switch e := expr.(type) {
    case *langlang.SequenceNode:
        ri.Kind = RuleSequence
        ri.Children = classifySequenceChildren(e)

    case *langlang.ChoiceNode:
        ri.Kind = RuleChoice
        ri.Choices = flattenChoices(e)

    case *langlang.ZeroOrMoreNode:
        ri.Kind = RuleRepeat
        if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
            ri.Inner = id.Value
        }

    case *langlang.OneOrMoreNode:
        ri.Kind = RuleRepeat
        if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
            ri.Inner = id.Value
        }

    case *langlang.OptionalNode:
        ri.Kind = RuleOptional
        if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
            ri.Inner = id.Value
        }

    case *langlang.IdentifierNode:
        ri.Kind = RuleAlias
        ri.Inner = e.Value

    case *langlang.LiteralNode, *langlang.ClassNode, *langlang.RangeNode,
        *langlang.CharsetNode, *langlang.AnyNode:
        ri.Kind = RuleLeaf

    default:
        // Complex expression we can't classify simply — treat as leaf
        ri.Kind = RuleLeaf
    }

    return ri
}

// unwrapTransparent strips AST wrappers that don't affect tree structure:
// LabeledNode (error recovery), LexNode (whitespace suppression),
// CaptureNode (already handled by VM).
func unwrapTransparent(node langlang.AstNode) langlang.AstNode {
    for {
        switch n := node.(type) {
        case *langlang.LabeledNode:
            node = n.Expr
        case *langlang.LexNode:
            node = n.Expr
        case *langlang.CaptureNode:
            node = n.Expr
        default:
            return node
        }
    }
}

// flattenChoices converts binary right-associative ChoiceNode into a flat
// list of alternative names. Non-identifier alternatives (literals) get
// an empty string entry.
func flattenChoices(c *langlang.ChoiceNode) []string {
    var choices []string
    var walk func(langlang.AstNode)
    walk = func(node langlang.AstNode) {
        node = unwrapTransparent(node)
        switch n := node.(type) {
        case *langlang.ChoiceNode:
            walk(n.Left)
            walk(n.Right)
        case *langlang.IdentifierNode:
            choices = append(choices, n.Value)
        default:
            // Literal or other non-rule alternative
            choices = append(choices, "")
        }
    }
    walk(c)
    return choices
}

// classifySequenceChildren inspects each item in a sequence and tags
// it as a literal (dead child) or named rule reference.
func classifySequenceChildren(seq *langlang.SequenceNode) []RuleChild {
    var children []RuleChild
    for i, item := range seq.Items {
        item = unwrapTransparent(item)
        rc := RuleChild{Index: i}

        switch n := item.(type) {
        case *langlang.IdentifierNode:
            rc.RuleName = n.Value
        case *langlang.LiteralNode, *langlang.ClassNode, *langlang.RangeNode,
            *langlang.CharsetNode, *langlang.AnyNode:
            rc.IsLiteral = true
        case *langlang.OptionalNode:
            // Optional wrapping a rule reference
            if id, ok := unwrapTransparent(n.Expr).(*langlang.IdentifierNode); ok {
                rc.RuleName = id.Value
            } else {
                rc.IsLiteral = true
            }
        case *langlang.ZeroOrMoreNode, *langlang.OneOrMoreNode:
            // Repetition — try to extract inner rule name
            var inner langlang.AstNode
            if zm, ok := n.(*langlang.ZeroOrMoreNode); ok {
                inner = zm.Expr
            } else if om, ok := n.(*langlang.OneOrMoreNode); ok {
                inner = om.Expr
            }
            if inner != nil {
                if id, ok := unwrapTransparent(inner).(*langlang.IdentifierNode); ok {
                    rc.RuleName = id.Value
                }
            }
        case *langlang.SequenceNode:
            // Nested sequence (e.g., from grouping) — treat as complex
            rc.IsLiteral = true
        default:
            rc.IsLiteral = true
        }
        children = append(children, rc)
    }
    return children
}
```

**Important:** `Bytecode.smap` is unexported. Before implementing
`resolveNameID`, check if `Program.StringID()` (at `go/vm_program.go:65`) works,
or add a public accessor to `Bytecode`. The `Program` approach uses
`QueryProgram` instead of `QueryBytecode`:

``` go
program, err := langlang.QueryProgram(db, grammarPath)
nameID := int32(program.StringID(def.Name))
```

If `Program.StringID` returns 0 for unknown names (ambiguous with the first
string), you may need to add a `Bytecode.StringID(name string) (int32, bool)`
method to `go/vm.go`. This would be a one-line addition:

``` go
func (b *Bytecode) StringID(name string) (int, bool) {
    id, ok := b.smap[name]
    return id, ok
}
```

This is the only modification to an existing langlang file. Decide during
implementation which approach is cleaner.

**Step 4: Run test to verify it passes**

Run: `cd go && go test ./extract/ -run TestAnalyzeGrammar -v` Expected: PASS
(may need to adjust based on nameID resolution approach)

**Step 5: Commit**

Message: `go[extract] add grammar analysis via QueryAST`

--------------------------------------------------------------------------------

### Task 4: Cross-validation

Merge struct analysis and grammar analysis. Reclassify fields using grammar
knowledge. Emit hard errors for mismatches.

**Promotion criteria:** N/A

**Files:** - Create: `go/extract/validate.go` - Create:
`go/extract/validate_test.go`

**Step 1: Write the failing test**

``` go
// go/extract/validate_test.go
package extract

import "testing"

func TestValidateUnknownRule(t *testing.T) {
    structs := []StructInfo{{
        Name: "Foo",
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
    // The parent struct should be reclassified as FieldChoice
    if result[0].Fields[0].Kind != FieldOptional {
        t.Errorf("expected fields to remain FieldOptional, got %d", result[0].Fields[0].Kind)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test ./extract/ -run TestValidate -v` Expected: FAIL

**Step 3: Write the implementation**

``` go
// go/extract/validate.go
package extract

import "fmt"

// Validate cross-checks struct fields against grammar rules. It reclassifies
// fields using grammar knowledge and returns errors for mismatches.
// The returned []StructInfo has NameID populated on each field.
func Validate(structs []StructInfo, rules map[string]RuleInfo) ([]StructInfo, []error) {
    var errs []error
    result := make([]StructInfo, len(structs))
    copy(result, structs)

    for i := range result {
        for j := range result[i].Fields {
            f := &result[i].Fields[j]

            rule, ok := rules[f.LLTag]
            if !ok {
                errs = append(errs, fmt.Errorf(
                    "%s.%s: unknown rule %q", result[i].Name, f.GoName, f.LLTag))
                continue
            }

            f.NameID = rule.NameID

            if err := validateFieldAgainstRule(f, &rule, &result[i]); err != nil {
                errs = append(errs, fmt.Errorf(
                    "%s.%s: %w", result[i].Name, f.GoName, err))
            }
        }
    }

    return result, errs
}

func validateFieldAgainstRule(f *FieldInfo, rule *RuleInfo, parent *StructInfo) error {
    switch f.Kind {
    case FieldText:
        // string field must map to a leaf or alias-to-leaf rule
        switch rule.Kind {
        case RuleLeaf, RuleAlias:
            return nil
        default:
            return fmt.Errorf(
                "string field mapped to %s rule %q (expected leaf or alias)",
                ruleKindString(rule.Kind), f.LLTag)
        }

    case FieldNamedRule:
        // struct field must map to a non-leaf rule
        switch rule.Kind {
        case RuleSequence, RuleChoice, RuleAlias, RuleRepeat:
            return nil
        case RuleLeaf:
            return fmt.Errorf(
                "struct field mapped to leaf rule %q", f.LLTag)
        }

    case FieldOptional:
        // pointer type — always valid, inner type determines behavior
        return nil

    case FieldSlice:
        // slice field — valid for repeat or sequence rules (where we
        // collect matching children by name)
        return nil

    case FieldChoice:
        // choice struct — verify alternatives exist
        if rule.Kind != RuleChoice {
            return fmt.Errorf(
                "choice struct mapped to non-choice rule %q", f.LLTag)
        }
    }

    return nil
}

func ruleKindString(k RuleKind) string {
    switch k {
    case RuleLeaf:
        return "leaf"
    case RuleSequence:
        return "sequence"
    case RuleChoice:
        return "choice"
    case RuleRepeat:
        return "repeat"
    case RuleOptional:
        return "optional"
    case RuleAlias:
        return "alias"
    default:
        return "unknown"
    }
}
```

**Step 4: Run test to verify it passes**

Run: `cd go && go test ./extract/ -run TestValidate -v` Expected: PASS

**Step 5: Commit**

Message: `go[extract] add cross-validation`

--------------------------------------------------------------------------------

### Task 5: Code emission

Generate extraction function bodies for each struct. Uses `fmt.Fprintf` to build
code strings per field kind --- same pattern as tommy's `emit.go`.

**Promotion criteria:** N/A

**Files:** - Create: `go/extract/emit.go` - Create: `go/extract/emit_test.go`

**Step 1: Write the failing test**

Test that emitted code for a simple choice struct contains the expected patterns
(nameID constants, switch cases, type assertions):

``` go
// go/extract/emit_test.go
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
        "Value": {Name: "Value", Kind: RuleChoice, NameID: 0,
            Choices: []string{"Object", "String"}},
        "Object": {Name: "Object", Kind: RuleSequence, NameID: 1},
        "String": {Name: "String", Kind: RuleLeaf, NameID: 3},
    }

    code := emitExtractFunction(si, rules, true)

    // Should contain nameID checks
    if !strings.Contains(code, "_nameID_Object") {
        t.Error("missing _nameID_Object reference")
    }
    if !strings.Contains(code, "_nameID_String") {
        t.Error("missing _nameID_String reference")
    }
    // Should contain the function signature
    if !strings.Contains(code, "ExtractJSONValue") {
        t.Error("missing ExtractJSONValue function")
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

    // Should skip literals and match by nameID
    if !strings.Contains(code, "NodeType_Node") {
        t.Error("missing NodeType_Node check")
    }
    if !strings.Contains(code, "t.Text(") {
        t.Error("missing t.Text() call for string field")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test ./extract/ -run TestEmit -v` Expected: FAIL

**Step 3: Write the implementation**

The emit code generates Go source as strings. Each function walks children and
dispatches based on `FieldKind`. The `exported` parameter controls whether the
function name is `Extract...` (public) or `extract...` (private).

``` go
// go/extract/emit.go
package extract

import (
    "fmt"
    "strings"
)

// emitExtractFunction generates a complete extraction function for one struct.
func emitExtractFunction(si StructInfo, rules map[string]RuleInfo, exported bool) string {
    var buf strings.Builder
    prefix := "extract"
    if exported {
        prefix = "Extract"
    }
    funcName := prefix + si.Name

    fmt.Fprintf(&buf, "func %s(t *tree, id NodeID) (%s, error) {\n", funcName, si.Name)
    fmt.Fprintf(&buf, "\tvar out %s\n", si.Name)

    // Detect if this struct represents a choice (all pointer fields)
    if isChoiceStruct(&si) {
        emitChoiceBody(&buf, si, rules)
    } else {
        emitSequenceBody(&buf, si, rules)
    }

    fmt.Fprintf(&buf, "\treturn out, nil\n")
    fmt.Fprintf(&buf, "}\n")
    return buf.String()
}

func emitChoiceBody(buf *strings.Builder, si StructInfo, rules map[string]RuleInfo) {
    fmt.Fprintf(buf, "\tchild, ok := t.Child(id)\n")
    fmt.Fprintf(buf, "\tif !ok {\n")
    fmt.Fprintf(buf, "\t\treturn out, fmt.Errorf(\"%s: no child\")\n", si.Name)
    fmt.Fprintf(buf, "\t}\n")
    fmt.Fprintf(buf, "\tcn := &t.nodes[child]\n\n")
    fmt.Fprintf(buf, "\tswitch {\n")

    for _, f := range si.Fields {
        switch f.Kind {
        case FieldOptional:
            innerType := strings.TrimPrefix(f.GoType, "*")
            if innerType == "string" {
                // *string — extract text
                fmt.Fprintf(buf, "\tcase cn.typ == NodeType_Node && cn.nameID == _nameID_%s:\n", f.LLTag)
                fmt.Fprintf(buf, "\t\ts := t.Text(child)\n")
                fmt.Fprintf(buf, "\t\tout.%s = &s\n", f.GoName)
            } else {
                // *SomeStruct — call nested extract
                fmt.Fprintf(buf, "\tcase cn.typ == NodeType_Node && cn.nameID == _nameID_%s:\n", f.LLTag)
                fmt.Fprintf(buf, "\t\tval, err := extract%s(t, child)\n", innerType)
                fmt.Fprintf(buf, "\t\tif err != nil { return out, err }\n")
                fmt.Fprintf(buf, "\t\tout.%s = &val\n", f.GoName)
            }
        }
    }

    // Fallback for bare NodeType_String (literal alternatives in choices)
    fmt.Fprintf(buf, "\tcase cn.typ == NodeType_String:\n")
    fmt.Fprintf(buf, "\t\t// literal alternative\n")
    fmt.Fprintf(buf, "\t}\n")
}

func emitSequenceBody(buf *strings.Builder, si StructInfo, rules map[string]RuleInfo) {
    // Walk all descendants, matching named nodes by nameID
    fmt.Fprintf(buf, "\tt.Visit(id, func(cid NodeID) bool {\n")
    fmt.Fprintf(buf, "\t\tcn := &t.nodes[cid]\n")
    fmt.Fprintf(buf, "\t\tif cid == id { return true }\n")
    fmt.Fprintf(buf, "\t\tif cn.typ != NodeType_Node { return true }\n")
    fmt.Fprintf(buf, "\t\tswitch cn.nameID {\n")

    for _, f := range si.Fields {
        switch f.Kind {
        case FieldText:
            fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
            fmt.Fprintf(buf, "\t\t\tout.%s = t.Text(cid)\n", f.GoName)
            fmt.Fprintf(buf, "\t\t\treturn false\n")

        case FieldNamedRule:
            fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
            fmt.Fprintf(buf, "\t\t\tval, err := Extract%s(t, cid)\n", f.GoType)
            fmt.Fprintf(buf, "\t\t\tif err == nil { out.%s = val }\n", f.GoName)
            fmt.Fprintf(buf, "\t\t\treturn false\n")

        case FieldSlice:
            elemType := f.ElemType
            fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
            fmt.Fprintf(buf, "\t\t\tval, err := Extract%s(t, cid)\n", elemType)
            fmt.Fprintf(buf, "\t\t\tif err == nil { out.%s = append(out.%s, val) }\n",
                f.GoName, f.GoName)
            fmt.Fprintf(buf, "\t\t\treturn false\n")

        case FieldOptional:
            innerType := strings.TrimPrefix(f.GoType, "*")
            fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
            if innerType == "string" {
                fmt.Fprintf(buf, "\t\t\ts := t.Text(cid)\n")
                fmt.Fprintf(buf, "\t\t\tout.%s = &s\n", f.GoName)
            } else {
                fmt.Fprintf(buf, "\t\t\tval, err := extract%s(t, cid)\n", innerType)
                fmt.Fprintf(buf, "\t\t\tif err == nil { out.%s = &val }\n", f.GoName)
            }
            fmt.Fprintf(buf, "\t\t\treturn false\n")
        }
    }

    fmt.Fprintf(buf, "\t\t}\n")
    fmt.Fprintf(buf, "\t\treturn true\n")
    fmt.Fprintf(buf, "\t})\n")
}
```

**Note:** This emit code is a starting point. The exact patterns will likely
need adjustment when testing against real JSON parse trees --- particularly how
sequences nest their children (the design doc flagged this as modest
uncertainty). Be prepared to iterate on the emit patterns during the integration
test (Task 7).

**Step 4: Run test to verify it passes**

Run: `cd go && go test ./extract/ -run TestEmit -v` Expected: PASS

**Step 5: Commit**

Message: `go[extract] add code emission`

--------------------------------------------------------------------------------

### Task 6: Template and orchestrator

The template renders the full output file (package declaration, imports, nameID
constants, extract functions). The orchestrator (`generate.go`) wires analyze
-\> grammar -\> validate -\> emit -\> template -\> format -\> write.

**Promotion criteria:** N/A

**Files:** - Create: `go/extract/template.go` - Create: `go/extract/generate.go`

**Step 1: Write the failing test**

``` go
// go/extract/generate_test.go (add to existing file or create)
package extract

import (
    "os"
    "path/filepath"
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
    if !strings.Contains(output, "_nameID_Value") {
        t.Error("missing nameID constant")
    }
    if !strings.Contains(output, "DO NOT EDIT") {
        t.Error("missing generated code header")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test ./extract/ -run TestRenderFile -v` Expected: FAIL

**Step 3: Write the template and orchestrator**

``` go
// go/extract/template.go
package extract

import (
    "bytes"
    "fmt"
    "go/format"
    "strings"
    "text/template"
)

type NameIDEntry struct {
    Name string
    ID   int32
}

type renderData struct {
    Package     string
    GrammarPath string
    NameIDs     []NameIDEntry
    Functions   string
}

const fileTmpl = `// Code generated by langlang extract; DO NOT EDIT.
// Grammar: {{.GrammarPath}}

package {{.Package}}

import "fmt"

var _ = fmt.Errorf

const (
{{- range .NameIDs}}
    _nameID_{{.Name}} int32 = {{.ID}}
{{- end}}
)

{{.Functions}}
`

// RenderFile produces a formatted Go source file containing nameID constants
// and extraction functions.
func RenderFile(
    pkg, grammarPath string,
    nameIDs []NameIDEntry,
    structs []StructInfo,
    rules map[string]RuleInfo,
) (string, error) {
    var funcs strings.Builder
    for i, si := range structs {
        exported := i == 0 // first struct gets exported Extract function
        funcs.WriteString(emitExtractFunction(si, rules, exported))
        funcs.WriteString("\n")
    }

    tmpl, err := template.New("extract").Parse(fileTmpl)
    if err != nil {
        return "", fmt.Errorf("parse template: %w", err)
    }

    var buf bytes.Buffer
    err = tmpl.Execute(&buf, renderData{
        Package:     pkg,
        GrammarPath: grammarPath,
        NameIDs:     nameIDs,
        Functions:   funcs.String(),
    })
    if err != nil {
        return "", fmt.Errorf("execute template: %w", err)
    }

    formatted, err := format.Source(buf.Bytes())
    if err != nil {
        return "", fmt.Errorf("gofmt: %w (source:\n%s)", err, buf.String())
    }

    return string(formatted), nil
}
```

``` go
// go/extract/generate.go
package extract

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// Generate is the main orchestrator. It reads a Go source file and grammar,
// cross-validates, and writes <source>_extract.go.
func Generate(sourceFile, grammarPath string) error {
    // Phase 1: Analyze Go structs
    structs, err := Analyze(sourceFile)
    if err != nil {
        return fmt.Errorf("analyze structs: %w", err)
    }
    if len(structs) == 0 {
        return fmt.Errorf("no structs with ll: tags found in %s", sourceFile)
    }

    // Phase 2: Analyze grammar
    rules, err := AnalyzeGrammar(grammarPath)
    if err != nil {
        return fmt.Errorf("analyze grammar: %w", err)
    }

    // Phase 3: Cross-validate
    structs, errs := Validate(structs, rules)
    if len(errs) > 0 {
        var msgs []string
        for _, e := range errs {
            msgs = append(msgs, e.Error())
        }
        return fmt.Errorf("validation errors:\n  %s", strings.Join(msgs, "\n  "))
    }

    // Collect nameID entries for all referenced rules
    nameIDs := collectNameIDs(structs, rules)

    // Detect package name from source file directory
    pkg := detectPackageName(sourceFile)

    // Phase 4: Render
    output, err := RenderFile(pkg, grammarPath, nameIDs, structs, rules)
    if err != nil {
        return fmt.Errorf("render: %w", err)
    }

    // Phase 5: Write output
    base := strings.TrimSuffix(filepath.Base(sourceFile), ".go")
    outPath := filepath.Join(filepath.Dir(sourceFile), base+"_extract.go")
    if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
        return fmt.Errorf("write %s: %w", outPath, err)
    }

    return nil
}

func collectNameIDs(structs []StructInfo, rules map[string]RuleInfo) []NameIDEntry {
    seen := map[string]bool{}
    var entries []NameIDEntry
    for _, si := range structs {
        for _, f := range si.Fields {
            if seen[f.LLTag] {
                continue
            }
            seen[f.LLTag] = true
            if rule, ok := rules[f.LLTag]; ok {
                entries = append(entries, NameIDEntry{Name: f.LLTag, ID: rule.NameID})
            }
        }
    }
    return entries
}

func detectPackageName(sourceFile string) string {
    // Read the first line of the source file for "package X"
    data, err := os.ReadFile(sourceFile)
    if err != nil {
        return "main"
    }
    for _, line := range strings.Split(string(data), "\n") {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "package ") {
            return strings.TrimSpace(strings.TrimPrefix(line, "package "))
        }
    }
    return "main"
}
```

**Step 4: Run test to verify it passes**

Run: `cd go && go test ./extract/ -run TestRenderFile -v` Expected: PASS

**Step 5: Commit**

Message: `go[extract] add template and orchestrator`

--------------------------------------------------------------------------------

### Task 7: Integration test --- JSON grammar

End-to-end test: generate extraction code for the JSON grammar, compile it,
parse real JSON, and verify extracted Go values.

This is the most important task --- it validates the entire pipeline and will
likely surface issues with emit patterns that need fixing.

**Promotion criteria:** N/A

**Files:** - Create: `go/extract/integration_test.go`

**Step 1: Write the integration test**

The test creates a temp directory, writes a Go source file with JSON extract
structs, runs `Generate()`, then compiles and tests the output. Since we can't
easily compile generated code within a test, we use a different approach: call
the extract pipeline, then manually verify the output file contains expected
patterns and compiles via `go/format`.

For actual runtime testing, create a test fixture in `go/examples/`:

``` go
// go/extract/integration_test.go
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

    // Write a source file with JSON extract structs
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

    // Run the generator
    err = Generate(src, grammarPath)
    if err != nil {
        t.Fatal(err)
    }

    // Verify output file exists
    outPath := filepath.Join(dir, "json_types_extract.go")
    data, err := os.ReadFile(outPath)
    if err != nil {
        t.Fatalf("output file not created: %v", err)
    }

    output := string(data)

    // Verify key patterns in generated code
    checks := []string{
        "DO NOT EDIT",
        "package json",
        "_nameID_Object",
        "_nameID_Array",
        "_nameID_String",
        "_nameID_Number",
        "_nameID_Member",
        "_nameID_Value",
        "ExtractJSONValue",
        "NodeType_Node",
        "NodeType_String",
    }
    for _, check := range checks {
        if !strings.Contains(output, check) {
            t.Errorf("output missing %q", check)
        }
    }

    // Log the output for debugging
    t.Logf("Generated output:\n%s", output)
}
```

**Step 2: Run test**

Run: `cd go && go test ./extract/ -run TestIntegrationJSON -v`

**Step 3: Iterate on emit patterns**

This test will likely surface issues. Fix them iteratively: - If the generated
code doesn't compile (gofmt fails), fix `emit.go` - If nameIDs aren't resolving,
fix the `resolveNameID` approach in `grammar.go` - If tree structure assumptions
are wrong, parse actual JSON with the grammar and inspect the tree output before
adjusting emit patterns

**Step 4: Commit**

Message: `go[extract] add JSON integration test`

--------------------------------------------------------------------------------

### Task 8: CLI subcommand

Add the `langlang extract` subcommand to the existing CLI.

**Promotion criteria:** N/A

**Files:** - Create: `go/cmd/langlang/extract.go` - Modify:
`go/cmd/langlang/main.go`

**Step 1: Write the extract subcommand**

``` go
// go/cmd/langlang/extract.go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/clarete/langlang/go/extract"
)

func runExtract(args []string) {
    fs := flag.NewFlagSet("extract", flag.ExitOnError)
    grammar := fs.String("grammar", "", "Path to the grammar file")
    fs.Parse(args)

    if *grammar == "" {
        fmt.Fprintf(os.Stderr, "error: -grammar is required\n")
        os.Exit(1)
    }

    // $GOFILE is set by go generate
    goFile := os.Getenv("GOFILE")
    if goFile == "" {
        fmt.Fprintf(os.Stderr, "error: $GOFILE not set (run via go generate)\n")
        os.Exit(1)
    }

    // Resolve relative to $GOFILE's directory
    if err := extract.Generate(goFile, *grammar); err != nil {
        fmt.Fprintf(os.Stderr, "error: %s\n", err)
        os.Exit(1)
    }
}
```

**Step 2: Add subcommand dispatch to main.go**

In `go/cmd/langlang/main.go`, add dispatch before the existing flag parsing.
Insert at the start of `main()`:

``` go
func main() {
    // Subcommand dispatch
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "extract":
            runExtract(os.Args[2:])
            return
        }
    }

    // Existing flag-based behavior
    a := readArgs()
    // ... rest of main unchanged
}
```

**Step 3: Test manually**

Create a test directory with a Go file and grammar, then run:

``` bash
cd go && go build -o /tmp/langlang ./cmd/langlang
cd /tmp/test-extract
GOFILE=types.go /tmp/langlang extract -grammar=json.peg
```

**Step 4: Commit**

Message: `go[extract] add CLI subcommand`

--------------------------------------------------------------------------------

### Task 9: Existing test suite --- no regressions

Verify all existing langlang tests still pass.

**Promotion criteria:** N/A

**Files:** - None (read-only verification)

**Step 1: Run full test suite**

Run: `cd go && go test ./... -v 2>&1 | head -100` Expected: All existing tests
PASS. The only change to existing files is the subcommand dispatch in `main.go`,
which is gated behind `os.Args[1] == "extract"` and shouldn't affect existing
behavior.

**Step 2: Run gofmt check**

Run: `cd go && gofmt -l extract/` Expected: No files listed (all formatted)

**Step 3: Commit (if any fixes needed)**

Message: `go[extract] fix any regressions`

--------------------------------------------------------------------------------

### Task 10: Runtime integration test (optional stretch)

If time permits, create a working example in `go/examples/json-extract/` that
actually parses JSON and extracts values at runtime.

**Promotion criteria:** N/A

**Files:** - Create: `go/examples/json-extract/json_types.go` - Create:
`go/examples/json-extract/json_types_extract.go` (generated) - Create:
`go/examples/json-extract/json_extract_test.go`

This task requires: 1. Generate the parser:
`langlang -grammar json.peg -output-language go ...` 2. Generate the extract
code: `langlang extract -grammar json.peg` 3. Write a test that parses
`{"name": "test", "count": 42}` and verifies the extracted struct values

This is a stretch goal because it requires coordinating parser codegen and
extract codegen in the same package. If the pipeline works, this is the ultimate
validation. If it doesn't compile, the error messages will guide what needs
fixing.

**Step 1: Set up the example directory and generate parser**

``` bash
mkdir -p go/examples/json-extract
cd go/examples/json-extract
# Generate parser from JSON grammar
go run ../../cmd/langlang \
  -grammar ../../../docs/live/assets/examples/json/json.peg \
  -output-language go \
  -output-path ./parser.go \
  -go-package jsonextract \
  -go-parser JSONParser
```

**Step 2: Write the types file with go:generate directive**

``` go
// go/examples/json-extract/json_types.go
package jsonextract

//go:generate go run ../../cmd/langlang extract -grammar=../../../docs/live/assets/examples/json/json.peg

type JSONValue struct {
    Object *JSONObject `ll:"Object"`
    Array  *JSONArray  `ll:"Array"`
    String *string     `ll:"String"`
    Number *string     `ll:"Number"`
}

type JSONObject struct {
    Members []JSONMember `ll:"Member"`
}

type JSONMember struct {
    Key   string    `ll:"String"`
    Value JSONValue `ll:"Value"`
}

type JSONArray struct {
    Items []JSONValue `ll:"Value"`
}
```

**Step 3: Generate extract code and write test**

``` bash
cd go/examples/json-extract
go generate ./...
```

``` go
// go/examples/json-extract/json_extract_test.go
package jsonextract

import "testing"

func TestExtractJSONObject(t *testing.T) {
    p := NewJSONParser()
    p.SetInput([]byte(`{"name": "test"}`))
    result, err := p.Parse()
    if err != nil {
        t.Fatal(err)
    }
    root, ok := result.Root()
    if !ok {
        t.Fatal("no root")
    }
    // Type assertion to access concrete tree
    tr := result.(*tree)
    val, err := ExtractJSONValue(tr, root)
    if err != nil {
        t.Fatal(err)
    }
    if val.Object == nil {
        t.Fatal("expected Object, got nil")
    }
    if len(val.Object.Members) != 1 {
        t.Fatalf("expected 1 member, got %d", len(val.Object.Members))
    }
    // Note: String extraction includes quotes — may need to strip them
    t.Logf("Key: %q, Value: %+v", val.Object.Members[0].Key, val.Object.Members[0].Value)
}
```

**Step 4: Run and iterate**

Run: `cd go && go test ./examples/json-extract/ -v`

This will likely need iteration. Common issues: - The `Parse()` root node might
be "JSON" not "Value" --- adjust extraction entry - String values include quotes
--- may need a `stripQuotes` helper - Tree nesting depth may differ from
assumptions --- inspect with `tree.Pretty()`

**Step 5: Commit**

Message: `go[extract] add JSON runtime integration example`
