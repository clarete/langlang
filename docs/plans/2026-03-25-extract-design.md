# Langlang Extract: Static Tree Extraction via go:generate

**Date:** 2026-03-25 **Status:** proposed **Scope:** JSON grammar
proof-of-concept

## Problem

Langlang parsers produce a `Tree` interface backed by an arena-allocated `tree`
struct. Consumers that want typed Go data must hand-write tree-walking code:
checking `Name()` against strings, navigating `Children()` by index, handling
optionals, and converting `Text()` to typed values. This boilerplate is
proportional to grammar complexity, error-prone (positional indexing is
fragile), and pays for interface dispatch on every node access even though the
grammar structure is statically known.

## Solution

A `langlang extract` subcommand invoked via `//go:generate` that reads Go struct
definitions with `ll:` tags and a PEG grammar file, then emits extraction
functions that index directly into the tree's arena slices. No interface
dispatch, no string comparisons for rule names, no positional guesswork.

## Scope

### In scope (JSON proof-of-concept)

- `go/extract/` package: struct analysis, grammar analysis, cross-validation,
  code emission
- `langlang extract` CLI subcommand
- Field kinds: FieldText, FieldNamedRule, FieldOptional, FieldSlice, FieldChoice
- Cross-validation at generation time (unknown rules, type/arity mismatches)
- Integration test against JSON grammar

### Deferred

- FieldSequenceChild, FieldCapture, FieldCustom (more complex grammars)
- Unmatched node tracking (TOML phase)
- Grammar-specific public tree types (post proof-of-concept, see below)
- `langlang types` (generate structs from grammar)
- Cross-package extraction
- Index-based sequence child mapping
- Encode path (structs -\> tree -\> text)
- `TreeExtractor` interface (TOML phase)
- LSP integration for `ll:` tag validation

## Architecture

    Consumer source file                    Grammar file
    +--------------------------+           +--------------+
    | //go:generate langlang   |           | json.peg     |
    |   extract -grammar=...   |           |              |
    |                          |           | Value <- ... |
    | type JSONValue struct {  |           | Object <- ...|
    |   Object *JSONObject     |           | ...          |
    |     `ll:"Object"`        |           +------+-------+
    |   ...                    |                  |
    | }                        |                  |
    +----------+---------------+                  |
               |                                  |
               v                                  v
         +-----------+                    +--------------+
         | analyze.go|                    | grammar.go   |
         | go/ast    |                    | QueryAST     |
         | -> []     |                    | QueryBytecode|
         | StructInfo|                    | -> map[]     |
         +-----+-----+                    |   RuleInfo   |
               |                          +------+-------+
               |          +-----------+          |
               +--------->|validate.go|<---------+
                          | cross-    |
                          | validate  |
                          +-----+-----+
                                |
                                v
                          +-----------+
                          | emit.go   |
                          | per-field |
                          | codegen   |
                          +-----+-----+
                                |
                                v
                          +-----------+
                          |template.go|
                          | file      |
                          | skeleton  |
                          +-----+-----+
                                |
                                v
                       json_types_extract.go

**Invocation:** `//go:generate langlang extract -grammar=json.peg`

The `$GOFILE` environment variable (set by `go generate`) tells the tool which
source file to analyze. The tool:

1.  **Analyzes structs** (`analyze.go`): Parses the source file with `go/ast`,
    finds structs with `ll:` tags, classifies fields by Go type.
2.  **Analyzes grammar** (`grammar.go`): Loads the `.peg` file via `QueryAST` +
    `QueryBytecode`, builds a `RuleInfo` map describing each rule's structure
    and nameID.
3.  **Cross-validates** (`validate.go`): Merges struct and grammar info,
    reclassifies fields using grammar knowledge, emits hard errors for
    mismatches.
4.  **Emits code** (`emit.go`): Generates arena-direct tree-walking code per
    field kind via `fmt.Fprintf`.
5.  **Renders file** (`template.go`): Wraps emitted code in file skeleton with
    nameID constants and function signatures.

**Output:** `<source>_extract.go` in the same directory.

## Type System

### StructInfo and FieldInfo

``` go
type StructInfo struct {
    Name   string
    Fields []FieldInfo
}

type FieldInfo struct {
    GoName   string      // Go field name
    LLTag    string      // rule name from ll:"..." tag
    Kind     FieldKind   // classification
    GoType   string      // Go type name
    ElemType string      // for slices: element type
    Inner    *StructInfo // for nested structs: recursive info
}
```

### FieldKind (proof-of-concept subset)

``` go
type FieldKind int

const (
    FieldText      FieldKind = iota // terminal rule -> string
    FieldNamedRule                  // rule reference -> struct with ll: tags
    FieldOptional                   // grammar ? or pointer choice branch
    FieldSlice                      // grammar * or + -> slice
    FieldChoice                     // grammar / -> struct with pointer fields
)
```

### RuleInfo (grammar side)

``` go
type RuleKind int

const (
    RuleLeaf     RuleKind = iota // terminal (literal, charset, range, any)
    RuleSequence                 // SequenceNode
    RuleChoice                   // ChoiceNode (possibly nested)
    RuleRepeat                   // ZeroOrMore or OneOrMore
    RuleOptional                 // OptionalNode
    RuleAlias                    // single IdentifierNode (rule reference)
)

type RuleInfo struct {
    Name     string
    Kind     RuleKind
    NameID   int32       // from Bytecode.smap
    Children []RuleChild // for sequences: ordered children
    Choices  []string    // for choices: alternative rule names
    Inner    string      // for alias/optional/repeat: referenced rule
}

type RuleChild struct {
    RuleName  string // empty for literals
    IsLiteral bool   // structural punctuation (dead child)
    Index     int    // position in SequenceNode.Items
}
```

### Cross-validation rules

  ---------------------------------------------------------------------------------------
  Go type          Rule kind                           Result           Error if
  ---------------- ----------------------------------- ---------------- -----------------
  `string`         RuleLeaf or RuleAlias-\>leaf        FieldText        rule is
                                                                        sequence/choice

  `SomeStruct`     RuleSequence/RuleChoice/RuleAlias   FieldNamedRule   rule is leaf
  (with `ll:`                                                           
  fields)                                                               

  `*SomeStruct`    any                                 FieldOptional    --

  `[]SomeStruct`   RuleRepeat                          FieldSlice       rule is not
                                                                        repeat

  struct with      RuleChoice                          FieldChoice      arity mismatch
  all-pointer                                                           
  `ll:` fields                                                          

  unknown rule     --                                  --               hard error
  name                                                                  
  ---------------------------------------------------------------------------------------

## Tree Structure Findings

Verified against the langlang codebase:

- **Literals in choices produce bare `NodeType_String`**, not `NodeType_Node`.
  For `Value <- Object / ... / 'true' / 'false' / 'null'`, the literal
  alternatives have no `nameID`. The emit code handles this with a
  `NodeType_String` fallback case.
- **Literals in sequences DO produce tree children.**
  `Member <- String ':' Value` produces a 3-child sequence. Dead child
  elimination skips `NodeType_String` children matching grammar literals.
- **LexNode (`#(...)`) is a no-op for tree structure.** Only suppresses
  whitespace insertion.
- **LabeledNode (`^label`) is purely error recovery.** No tree impact.
- **ChoiceNode is binary right-associative.** `A / B / C` becomes
  `ChoiceNode{A, ChoiceNode{B, C}}`. Grammar analysis flattens this into a list.

**Modest uncertainty:** The exact emit pattern for literal alternatives in
choices may need adjustment during implementation once tested against real tree
output.

## Generated Code Shape

### Consumer writes

``` go
//go:generate langlang extract -grammar=json.peg

type JSONValue struct {
    Object *JSONObject `ll:"Object"`
    Array  *JSONArray  `ll:"Array"`
    String *string     `ll:"String"`
    Number *string     `ll:"Number"`
    Raw    *string     // no ll tag -- captures literal alternatives
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

### Generator produces `json_types_extract.go`

``` go
// Code generated by langlang extract; DO NOT EDIT.
// Grammar: json.peg

package json

import "fmt"

var _ = fmt.Errorf

const (
    _nameID_Value  int32 = /* from bytecode smap */
    _nameID_Object int32 = /* ... */
    _nameID_Array  int32 = /* ... */
    _nameID_String int32 = /* ... */
    _nameID_Number int32 = /* ... */
    _nameID_Member int32 = /* ... */
)

func ExtractJSONValue(t *tree, id NodeID) (JSONValue, error) {
    var out JSONValue
    n := &t.nodes[id]

    child, ok := t.Child(id)
    if !ok {
        return out, fmt.Errorf("Value: no child")
    }
    cn := &t.nodes[child]

    switch {
    case cn.typ == NodeType_Node && cn.nameID == _nameID_Object:
        val, err := extractJSONObject(t, child)
        if err != nil { return out, err }
        out.Object = &val
    case cn.typ == NodeType_Node && cn.nameID == _nameID_Array:
        val, err := extractJSONArray(t, child)
        if err != nil { return out, err }
        out.Array = &val
    case cn.typ == NodeType_Node && cn.nameID == _nameID_String:
        s := t.Text(child)
        out.String = &s
    case cn.typ == NodeType_Node && cn.nameID == _nameID_Number:
        s := t.Text(child)
        out.Number = &s
    case cn.typ == NodeType_String:
        s := string(t.input[cn.start:cn.end])
        out.Raw = &s
    }

    return out, nil
}

func extractJSONObject(t *tree, id NodeID) (JSONObject, error) {
    var out JSONObject
    t.Visit(id, func(cid NodeID) bool {
        cn := &t.nodes[cid]
        if cn.typ == NodeType_Node && cn.nameID == _nameID_Member {
            val, err := extractJSONMember(t, cid)
            if err == nil {
                out.Members = append(out.Members, val)
            }
            return false
        }
        return true
    })
    return out, nil
}

func extractJSONMember(t *tree, id NodeID) (JSONMember, error) {
    var out JSONMember
    for _, cid := range t.Children(id) {
        for _, sid := range t.Children(cid) {
            sn := &t.nodes[sid]
            switch {
            case sn.typ == NodeType_Node && sn.nameID == _nameID_String:
                out.Key = t.Text(sid)
            case sn.typ == NodeType_Node && sn.nameID == _nameID_Value:
                val, err := ExtractJSONValue(t, sid)
                if err != nil { return out, err }
                out.Value = val
            }
        }
    }
    return out, nil
}
```

Key properties of the generated code:

- Name checks use `nameID` integer constants, not string comparison
- Literals are skipped by checking `NodeType_String`
- Choice branches dispatch on `nameID` or fall back to `NodeType_String` for
  literal alternatives
- No `Tree` interface dispatch -- direct `t.nodes[id]` access
- Type assertion `result.(*tree)` used at entry point (see decisions)

## Project Structure

    go/
    +-- extract/
    |   +-- analyze.go          # go/ast -> []StructInfo
    |   +-- analyze_test.go
    |   +-- grammar.go          # QueryAST + QueryBytecode -> map[string]RuleInfo
    |   +-- grammar_test.go
    |   +-- validate.go         # cross-validation, field reclassification
    |   +-- validate_test.go
    |   +-- emit.go             # per-field codegen via fmt.Fprintf
    |   +-- emit_test.go
    |   +-- template.go         # text/template file skeleton
    |   +-- generate.go         # orchestrator
    |   +-- generate_test.go    # integration test against JSON grammar
    +-- cmd/
        +-- langlang/
            +-- main.go         # modified: add extract subcommand dispatch
            +-- extract.go      # new: extract subcommand entry point

### Changes to existing files

- `go/cmd/langlang/main.go`: Add subcommand dispatch via `flag.Args()`.

No changes to `gen.go`, `api.go`, `tree.go`, or any other existing file.

### Dependencies

- `go/ast`, `go/parser`, `go/token`, `go/format` -- stdlib, already used
- `text/template` -- stdlib, already used
- `langlang` package -- for `QueryAST`, `QueryBytecode`, grammar AST types

No new external dependencies.

## Decisions

  --------------------------------------------------------------------------------
  Decision                           Rationale
  ---------------------------------- ---------------------------------------------
  Tag name: `ll`                     Mirrors tommy's `toml` convention;
                                     langlang-specific. Easy to rename later.

  Concrete tree access: type         `result.(*tree)` in generated extract entry
  assertion                          point. Zero changes to gen.go. Ideally
                                     `MatchRule`/`parseFn` would return concrete
                                     types directly; flagged for future cleanup so
                                     no type assertions are needed.

  Sequence child mapping: name-based `ll:"String"` matches child by rule name.
                                     Sufficient for JSON and likely TOML.
                                     Index-based escape hatch deferred for
                                     grammars with duplicate rule names in
                                     sequences (e.g., `Expr Op Expr`). A more
                                     ergonomic mapping strategy beyond raw
                                     indexing should also be explored.

  Struct analysis: `go/ast` only     Lightweight, no `go/packages` dependency.
                                     **Pivot path:** if single-file resolution
                                     proves limiting (struct field references type
                                     in another file), add
                                     `golang.org/x/tools/go/packages` and replace
                                     the `go/ast` parse with `packages.Load` using
                                     `packages.NeedTypes \| packages.NeedSyntax`
                                     mode. The rest of the pipeline stays the
                                     same.

  Same-package extraction only       Extract code and parser live in same
                                     generated package. Cross-package deferred.

  Proof-of-concept grammar: JSON     Well-known, exercises
                                     choice/sequence/repetition/optional. Next
                                     target: TOML.
  --------------------------------------------------------------------------------

## Rollback

Purely additive. Delete `_extract.go` files and remove
`//go:generate langlang extract` directives. The `Tree` interface path remains
fully functional. No existing consumer is affected.

## Implemented Optimizations

### `UnsafeText` (zero-copy string extraction)

`tree.UnsafeText(id) string` returns a string pointing directly into the parse
input buffer via `unsafe.String`, avoiding the `string([]byte)` copy in
`Text()`. Generated extract code uses `UnsafeText` instead of `Text`.

**Why it's safe despite using `unsafe`:**

1.  `t.input` is never mutated after parsing --- the VM writes it once during
    `bindInput()`, then the tree is read-only.
2.  The returned string cannot outlive the tree --- extract functions take
    `*tree` as a parameter, so the caller holds a reference.
3.  The input buffer is caller-owned and retained by the tree.

The compiler can't see through the `*tree` abstraction to prove these
invariants, so `string([]byte)` conservatively copies. `UnsafeText` makes the
programmer's knowledge explicit. The safe `Text()` method remains for callers
who need strings that outlive the tree.

**Impact (TOML benchmarks):** 20% fewer allocations in arena-direct extraction.
Combined with nameID constants, arena-direct eliminates 52-53% of allocations vs
the Tree interface approach.

## Generation Modes

The extract tool supports (or will support) multiple generation modes for
different consumer needs:

### `-mode=structs` (current, default)

Generates owned, mutable Go structs populated by walking the arena. Structs are
independent of the tree after extraction --- callers can modify fields,
serialize, or pass them across API boundaries.

**Best for:** read-write document editing (tommy), data transformation,
serialization, any case where the extracted data outlives the parse tree.

**Cost:** allocation per struct + per string field (mitigated by `UnsafeText`
for strings, but struct/slice allocations remain).

### `-mode=views` (planned)

Generates zero-allocation view types that wrap `*tree` + `NodeID`. Views provide
typed, read-only access to the parse tree without copying any data. Method calls
navigate the arena directly.

``` go
// Generated from grammar
type ValueView struct { t *tree; id NodeID }

func (v ValueView) Object() (ObjectView, bool) {
    child, ok := v.t.Child(v.id)
    if !ok || !v.t.IsNamed(child, _nameID_Object) {
        return ObjectView{}, false
    }
    return ObjectView{t: v.t, id: child}, true
}

func (v ValueView) String() (string, bool) {
    child, ok := v.t.Child(v.id)
    if !ok || !v.t.IsNamed(child, _nameID_String) {
        return "", false
    }
    return v.t.UnsafeText(child), true
}
```

**Best for:** read-only consumers (config parsing, query evaluation, validation,
tree-sitter-style navigation), performance-critical paths where allocation
pressure matters.

**Cost:** views are only valid while the tree is alive. No mutation, no
serialization without copying first.

**Design notes:**

- Choice rules return `(XxxView, bool)` per alternative --- caller checks which
  matched
- Sequence rules expose named children as methods
- Repetition rules return an iterator or `Visit`-style callback
- Leaf rules return `(string, bool)` via `UnsafeText`
- View types are value types (2 words: pointer + int) --- free to copy, compare,
  pass by value

### Full parser generation (future)

A mode where the parser emits typed structs directly during parsing, with no
intermediate generic arena. The grammar + struct definitions are known at
generation time, so the parser can write directly into typed fields.

**Challenges:** repetition (unknown count upfront requires growing slices or
slab allocation), sequences (must handle partial parse failures and
backtracking).

**Deferred** until the extract pipeline and view mode are proven.

## Future Work

### TOML phase

- Unmatched node tracking (`visited []bool` + `Unmatched()` method) for
  detecting grammar/struct drift
- `TreeExtractor` interface for custom extraction logic
- FieldCapture, FieldSequenceChild kinds

### Later

- Index-based sequence child mapping (and more ergonomic alternatives)
- `langlang types` -- generate Go structs directly from grammar
- Encode path (structs -\> tree -\> text)
- Cross-package extraction
- LSP integration for `ll:` tag validation and completion
- Eliminate type assertion by having `parseFn` return `*tree` directly
