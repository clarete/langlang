package langlang

// TreeBuilder provides a public API for constructing parse trees.
// This enables external packages (examples, tests, code generators)
// to build tree inputs for the rewrite system.
type TreeBuilder struct {
	T *tree
}

// NewTreeBuilder creates a fresh, empty tree builder.
func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{
		T: &tree{
			input: []byte{},
			strs:  []string{},
		},
	}
}

// Tree returns the constructed tree as a Tree interface.
func (b *TreeBuilder) Tree() Tree {
	return b.T
}

// Intern adds a string to the tree's string table and returns its ID.
func (b *TreeBuilder) Intern(s string) int32 {
	for i, existing := range b.T.strs {
		if existing == s {
			return int32(i)
		}
	}
	id := int32(len(b.T.strs))
	b.T.strs = append(b.T.strs, s)
	return id
}

// Str creates a string node containing the given text.
func (b *TreeBuilder) Str(text string) NodeID {
	start := len(b.T.input)
	b.T.input = append(b.T.input, []byte(text)...)
	end := len(b.T.input)
	return b.T.AddString(start, end)
}

// Named creates a named node wrapping a single child.
func (b *TreeBuilder) Named(name string, child NodeID) NodeID {
	return b.T.AddNode(b.Intern(name), child, 0, 0)
}

// Seq creates a sequence node from the given children.
func (b *TreeBuilder) Seq(children ...NodeID) NodeID {
	return b.T.AddSequence(children, 0, 0)
}

// Node creates a named node. With one child it wraps directly;
// with multiple children it wraps a sequence.
func (b *TreeBuilder) Node(name string, children ...NodeID) NodeID {
	if len(children) == 1 {
		return b.Named(name, children[0])
	}
	return b.Named(name, b.Seq(children...))
}
