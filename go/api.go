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
	Match([]byte) (Tree, int, error)
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
// The tree does not copy matched text. Each node records a Range
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

	// Range returns the byte offsets (start inclusive, end
	// exclusive) into the original input that this node spans.
	Range(NodeID) Range

	// Name returns the grammar rule name for NodeType_Node and
	// the error label for NodeType_Error nodes. Returns an empty
	// string for other node types.
	Name(NodeID) string

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
}

// MatcherFromBytes creates a Matcher from a PEG grammar definition
// provided as a byte slice.  The function parses the grammar,
// compiles it into assembly instructions, encodes it into bytecode,
// and returns a VirtualMachine ready to match input against the
// grammar rules.
//
// The cfg parameter controls various grammar transformations such as
// adding built-in rules, character set optimizations, whitespace
// handling, and capture generation.
//
// Returns an error if the grammar cannot be parsed, compiled, or if
// any transformation fails.
func MatcherFromBytes(input []byte, cfg *Config) (Matcher, error) {
	ast, err := GrammarFromBytes(input, cfg)
	if err != nil {
		return nil, err
	}
	asm, err := Compile(ast, cfg)
	if err != nil {
		return nil, err
	}
	code := Encode(asm)
	return NewVirtualMachine(code, nil, true), nil
}

// MatcherFromFile creates a Matcher from a PEG grammar file at the
// specified path. The function loads the grammar file (resolving any
// imports), parses it, compiles it into assembly instructions,
// encodes it into bytecode, and returns a VirtualMachine ready to
// match input against the grammar rules.
//
// The cfg parameter controls various grammar transformations such as
// adding built-in rules, character set optimizations, whitespace
// handling, and capture generation.
//
// Returns an error if the file cannot be read, the grammar cannot be
// parsed or compiled, or if any transformation fails.
func MatcherFromFile(path string, cfg *Config) (Matcher, error) {
	ast, err := GrammarFromFile(path, cfg)
	if err != nil {
		return nil, err
	}
	asm, err := Compile(ast, cfg)
	if err != nil {
		return nil, err
	}
	code := Encode(asm)
	return NewVirtualMachine(code, nil, true), nil
}
