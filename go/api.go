package langlang

// Matcher provides a dynamic PEG parsing interface created at runtime
// from a grammar definition.  Unlike ahead-of-time code generation,
// Matchers are built on-the-fly by compiling PEG grammars into
// bytecode and creating a virtual machine for running it.
type Matcher interface {
	// Match attempts to parse the input data against the
	// grammar's start rule, returning a syntax tree (Tree), the
	// number of bytes consumed, and any error encountered in the
	// process.
	//
	// NOTE: the Tree returned by the matcher is *borrowed* from
	// the Matcher.  So if you want to mutate it or use it after
	// calling [Matcher.Match] again, refer to the [Tree.Copy]
	// method.
	Match([]byte) (Tree, int, error)

	// SourceMap returns the optional mapping between the
	// matcher's bytecode and its original text source.  Among
	// other features, this enables tracking what expression
	// within the grammar failed causing the parser to stop.
	SourceMap() *SourceMap
}

// NodeID is an opaque handle identifying a node within a Tree.
type NodeID uint32

// NodeType discriminates the four kinds of nodes in a parse tree.
type NodeType uint8

const (
	// NodeType_String: a terminal match storing a byte range
	NodeType_String NodeType = iota
	// NodeType_Sequence: an ordered list of child nodes
	NodeType_Sequence
	// NodeType_Node: a named rule match with a single child
	NodeType_Node
	// NodeType_Error: a recovery point containing error metadata
	NodeType_Error
)

// Location identifies a position in an input source.
//
// Cursor is a byte offset into the input (0-based). Line and Column
// are 1-based and are computed in terms of UTF-8 decoded runes (not
// bytes).
//
// FileID is optional metadata (e.g. an interned filename id). When
// unavailable, it is set to -1.
type Location struct {
	Line   int
	Column int
	Cursor int
}

// Span represents a half-open interval [Start.Cursor, End.Cursor) in
// the input.
type Span struct {
	Start Location
	End   Location
}

// Tree represents a parse tree produced by matching input against a
// PEG grammar.  The tree structure is immutable once created. Nodes
// are accessed by NodeID, an opaque handle returned by Root() and
// navigation methods.
//
// A tree contains four node types (see NodeType):
//
//   - NodeType_String: a terminal match storing a byte range
//   - NodeType_Sequence: an ordered list of child nodes
//   - NodeType_Node: a named rule match with a single child
//   - NodeType_Error: a recovery point containing error metadata
//
// The tree does not copy matched text.  Each node records a range
// (start/end byte offsets) referencing the original input.
//
// Example usage:
//
//	tree, _, _ := matcher.Match(input)
//	root, _ := tree.Root()
//	fmt.Println(tree.Name(root))        // rule name
//	fmt.Println(tree.Text(root))        // matched text
//	for _, child := range tree.Children(root) {
//	    fmt.Println(tree.Text(child))
//	}
//
// Note on memory ownership: A [Matcher.Match] call returns a
// *borrowed* output [Tree] that will be reused on the very next call
// to [Matcher.Match]. If you want to keep using a tree after another
// match (or if you want to mutate it), use the [Tree.Copy] method.
// Here are a couple examples:
//
// 1. You don't need to copy because a) you're only reading the tree, and
// b) you're reading the tree *before* [Matcher.Match] is called
// again:
//
//	db := NewDatabase(NewConfig(), NewRelativeImportLoader())
//	matcher, _ := QueryMatcher(db, "main.peg")
//	for _, input := range inputs {
//	    tree, _, _ := matcher.Match(input)
//	    yourFunctionThatOnlyReadsTheTree(tree)
//	}
//
// 2. You do need to copy because you're reading the tree after
// [Matcher.Match] is called again:
//
//	db := NewDatabase(NewConfig(), NewRelativeImportLoader())
//	matcher, _ := QueryMatcher(db, "main.peg")
//	trees := make([]Tree, 0, len(inputs))
//	for _, input := range inputs {
//	    tree, _, _ := matcher.Match(input)
//	    trees = append(trees, tree.Copy())
//	}
//	yourFunctionThatReadsTreesAfterMatching(trees)
//
// Also, the tree borrows the `input` received by the [Matcher.Match]
// call that generated it as well as the string table of the
// [Matcher], so the [Matcher] and the parsed input will always
// outlive the tree.
type Tree interface {
	// Root returns the top-level node of the parse tree. The bool
	// is false if the tree is empty.
	Root() (NodeID, bool)

	// Visit visits all nodes in the parse tree in depth-first
	// order.  The function `fn` is called for each node under
	// `id`.  If `fn` returns false, the traversal is aborted.
	Visit(id NodeID, fn func(NodeID) bool)

	// Type returns the NodeType of the given node, indicating
	// whether it is a string literal, sequence, named node, or
	// error.
	Type(NodeID) NodeType

	// Span returns the node span as a pair of Locations (Start/End) where cursor
	// offsets are byte-based and line/column are 1-based UTF-8 rune columns.
	Span(NodeID) Span

	// Location converts an arbitrary cursor byte offset into a Location using the
	// same indexing as Span. Cursor is a byte offset; Column is rune-based.
	Location(cursor int) Location

	// CursorU16 converts a cursor byte offset (into the original UTF-8 input) into
	// an absolute UTF-16 code-unit offset. This is useful for consumers that use
	// UTF-16 indexing (e.g. Monaco, many LSP clients).
	CursorU16(cursor int) int

	// Name returns the grammar rule name for NodeType_Node and
	// the error label for NodeType_Error nodes. Returns an empty
	// string for other node types.
	Name(NodeID) string

	// Message returns the message for NodeType_Error nodes. Returns an empty
	// string for other node types.
	Message(NodeID) string

	// Child returns the single child of a NodeType_Node or
	// NodeType_Error. The bool is false for other node types or
	// if the node has no child.
	Child(NodeID) (NodeID, bool)

	// Children returns all direct children of a node. For
	// NodeType_Sequence, this is the list of child nodes. For
	// NodeType_Node and NodeType_Error, returns a single-element
	// slice. Returns nil for NodeType_String or childless nodes.
	Children(NodeID) []NodeID

	// Text extracts the matched substring from the original
	// input.  For sequences, it concatenates all descendant text.
	Text(NodeID) string

	// Pretty returns a human-readable, indented representation of
	// the subtree rooted at the given node, showing node types,
	// names, and byte ranges.
	Pretty(NodeID) string

	// Highlight is like Pretty but adds ANSI color codes for
	// terminal display.
	Highlight(NodeID) string

	// Copy allows users of the tree to take their own copy of the
	// result returned by the Matcher, which is originally
	// *borrowed* from the matcher.
	Copy() Tree
}

// StopReason describes why a debug session stopped
type StopReason uint8

const (
	StopNone StopReason = iota
	StopBreakpoint
	StopStep
	StopTerminated
)

// Stop describes a debugger stop point
type Stop struct {
	Reason StopReason
	PC     int
	Cursor int
}

// StackFrameType distinguishes between call, backtracking, and capture frames.
type StackFrameType int8

const (
	StackFrameBacktrack StackFrameType = iota
	StackFrameCall
	StackFrameCapture
)

// StackFrame represents a stack frame in the VM.
// This is a superset of fields needed by both the debugger and oracle:
//   - Debugger uses: ID, PC, Cursor, CursorU16, Name
//   - Oracle uses: PC, Cursor, Type, Predicate
type StackFrame struct {
	ID        int            // debugger: frame sequence number
	PC        int            // both: program counter / return address
	Cursor    int            // both: input byte position
	CursorU16 int            // debugger: UTF-16 offset for LSP
	Name      string         // debugger: rule name
	Type      StackFrameType // oracle: backtrack/call/capture
	Predicate bool           // oracle: true if inside a Not predicate
}

// DebugSession provides a stepping interface for the VM.
type DebugSession interface {
	// SetBreakpoint sets a breakpoint from a grammar location
	SetBreakpoint(Location)

	// ClearBreakpoint removes a breakpoint at the given location
	ClearBreakpoint(Location)

	// ListBreakpoints returns all currently set breakpoints as
	// locations
	ListBreakpoints() []Location

	// ClearBreakpoints removes all breakpoints
	ClearBreakpoints()

	// Continue runs Match until the next breakpoint or
	// termination.
	Continue() (Stop, error)

	// StepIn steps into the next Production call.
	StepIn() (Stop, error)

	// StepOver steps over the next Production call.
	StepOver() (Stop, error)

	// StepOut steps out of the current Production
	StepOut() (Stop, error)

	// StackTrace returns the current stack trace.
	StackTrace(limit int) []StackFrame

	// State returns the current PC and cursor position.
	State() (pc int, cursor int, terminated bool)
}

// ImportLoader abstracts how grammars are located and read during
// import resolution.
//
// The import system is where langlang assembles a complete grammar
// from an entrypoint plus its imports.  We currently have two different
// implementations of ImportLoader:
//
// - [NewRelativeImportLoader]: loads grammars from the filesystem
// - [NewInMemoryImportLoader]: loads grammars from memory
//
// Example (filesystem loader):
//
//	loader := NewRelativeImportLoader()
//
// Example (in-memory loader, useful for wasm/tests):
//
//	loader := NewInMemoryImportLoader()
//	loader.Add("value.peg", []byte(`Value <- [0-9]+`))
//	loader.Add("main.peg", []byte(`
//	   @import Value from "./value.peg"
//	   Main <- Value '+' ValueEOF^eof
//	`))
//
// And then we can use the query system:
//
//	db := NewDatabase(NewConfig(), loader)
//	ast, err := QueryAST(db, "main.peg")
type ImportLoader interface {
	// GetPath resolves an import path (as written in the grammar,
	// e.g.  "./expr.peg") relative to the parent module.  It
	// should return a stable module identifier that can later be
	// used to load content and to report spans/locations.
	GetPath(importPath, parentPath string) (string, error)

	// GetContent returns the bytes for a resolved path
	GetContent(path string) ([]byte, error)
}

// FileID is an interned identifier for a grammar source file
//
// When compiling grammars via the import resolver, paths are assigned
// a sequence of IDs.  This enables downstream components (compiler,
// bytecode, VM) to reference source files compactly and optionally
// map FileID(i) back to a resolved path via a side table.
//
// A value of -1 means the file is unknown or not applicable.
type FileID int

// SourceLocation identifies a span within a particular source file
type SourceLocation struct {
	FileID FileID
	Span   Span
}

// DiagnosticSeverity indicates the severity of a diagnostic.
type DiagnosticSeverity int

const (
	DiagnosticError DiagnosticSeverity = iota
	DiagnosticWarning
	DiagnosticInfo
	DiagnosticHint
)

// Diagnostic represents an error, warning, or informational message.
type Diagnostic struct {
	Location SourceLocation
	Severity DiagnosticSeverity
	Message  string
	Code     string    // e.g., "undefined-rule", "unused-rule"
	FilePath string    // the file path where the diagnostic occurred
	Expected []ErrHint // optional: what the parser expected (for syntax errors)
}

// SourceMap provides a mapping from bytecode offsets to grammar
// source locations. Data is stored in a delta + varint encoded binary
// format for compact storage. The data is decoded on first access and
// cached for subsequent lookups.
//
// Encoding format:
//   - First entry: absolute values using unsigned varints
//   - Subsequent entries: signed varints (zigzag) for deltas
//   - Fields per entry: Offset, FileID, StartLine, StartCol,
//     StartCursor, EndLine, EndCol, EndCursor
type SourceMap struct {
	Data    []byte        // Delta + varint encoded entries
	Files   []string      // FileID to Path/URI
	entries []srcMapEntry // cached decoded entries (lazily initialized)
}
