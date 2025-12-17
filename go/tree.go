package langlang

import (
	"fmt"
	"strconv"
	"strings"
)

type FormatToken int

const (
	FormatToken_None FormatToken = iota
	FormatToken_Range
	FormatToken_Literal
	FormatToken_Error
)

func (nt NodeType) String() string {
	switch nt {
	case NodeType_String:
		return "string"
	case NodeType_Sequence:
		return "sequence"
	case NodeType_Node:
		return "node"
	case NodeType_Error:
		return "error"
	default:
		return "unknown"
	}
}

type node struct {
	typ       NodeType
	start     int
	end       int
	nameID    int32
	childID   int32
	messageID int32
}

type tree struct {
	nodes       []node
	children    []NodeID
	childRanges []struct{ start, end int32 }
	strs        []string
	input       []byte
	root        NodeID
}

func newTree() *tree {
	return &tree{
		nodes:       make([]node, 0, 256),
		children:    make([]NodeID, 0, 512),
		childRanges: make([]struct{ start, end int32 }, 0, 256),
	}
}

func (t *tree) bindInput(input []byte)    { t.input = input }
func (t *tree) bindStrings(strs []string) { t.strs = strs }
func (t *tree) reset() {
	t.nodes = t.nodes[:0]
	t.children = t.children[:0]
	t.childRanges = t.childRanges[:0]
}

func (t *tree) Root() (NodeID, bool)                { return t.root, len(t.nodes) > 0 }
func (t *tree) SetRoot(id NodeID)                   { t.root = id }
func (t *tree) Type(id NodeID) NodeType             { return t.nodes[id].typ }
func (t *tree) MessageID(id NodeID) int32           { return t.nodes[id].messageID }
func (t *tree) NameID(id NodeID) int32              { return t.nodes[id].nameID }
func (t *tree) Name(id NodeID) string               { return t.strs[t.NameID(id)] }
func (t *tree) IsType(id NodeID, typ NodeType) bool { return t.Type(id) == typ }
func (t *tree) IsNamed(id NodeID, nameID int32) bool {
	return t.Type(id) == NodeType_Node && t.NameID(id) == nameID
}

func (t *tree) Range(id NodeID) Range {
	n := &t.nodes[id]
	return Range{Start: n.start, End: n.end}
}

func (t *tree) Children(id NodeID) []NodeID {
	n := &t.nodes[id]
	if n.childID == -1 {
		return nil
	}
	if n.typ == NodeType_Node || n.typ == NodeType_Error {
		return []NodeID{NodeID(n.childID)}
	}
	if n.typ == NodeType_Sequence {
		cr := t.childRanges[n.childID]
		return t.children[cr.start:cr.end]
	}
	return nil
}

func (t *tree) Child(id NodeID) (NodeID, bool) {
	childID := t.nodes[id].childID
	if childID == -1 {
		return 0, false
	}
	return NodeID(childID), true
}

func (t *tree) AddString(start, end int) NodeID {
	id := NodeID(len(t.nodes))
	t.nodes = append(t.nodes, node{
		typ:       NodeType_String,
		start:     start,
		end:       end,
		nameID:    -1,
		childID:   -1,
		messageID: -1,
	})
	return id
}

func (t *tree) AddSequence(children []NodeID, start, end int) NodeID {
	id := NodeID(len(t.nodes))
	childRangeID := int32(-1)
	if len(children) > 0 {
		childRangeID = int32(len(t.childRanges))
		childStart := int32(len(t.children))
		t.children = append(t.children, children...)
		childEnd := int32(len(t.children))
		t.childRanges = append(t.childRanges, struct{ start, end int32 }{childStart, childEnd})
	}
	t.nodes = append(t.nodes, node{
		typ:       NodeType_Sequence,
		start:     start,
		end:       end,
		childID:   childRangeID,
		nameID:    -1,
		messageID: -1,
	})
	return id
}

func (t *tree) AddNode(nameID int32, child NodeID, start, end int) NodeID {
	id := NodeID(len(t.nodes))
	t.nodes = append(t.nodes, node{
		typ:       NodeType_Node,
		start:     start,
		end:       end,
		nameID:    nameID,
		childID:   int32(child),
		messageID: -1,
	})
	return id
}

func (t *tree) AddError(labelID, messageID int32, start, end int) NodeID {
	id := NodeID(len(t.nodes))
	t.nodes = append(t.nodes, node{
		typ:       NodeType_Error,
		start:     start,
		end:       end,
		nameID:    labelID,
		childID:   -1,
		messageID: messageID,
	})
	return id
}

func (t *tree) AddErrorWithChild(labelID, messageID int32, childID NodeID, start, end int) NodeID {
	id := NodeID(len(t.nodes))
	t.nodes = append(t.nodes, node{
		typ:       NodeType_Error,
		start:     start,
		end:       end,
		nameID:    labelID,
		childID:   int32(childID),
		messageID: messageID,
	})
	return id
}

func (t *tree) Visit(id NodeID, fn func(NodeID) bool) {
	if !fn(id) {
		return
	}
	switch t.nodes[id].typ {
	case NodeType_Sequence:
		for _, child := range t.Children(id) {
			t.Visit(child, fn)
		}
	case NodeType_Node, NodeType_Error:
		if child, ok := t.Child(id); ok {
			t.Visit(child, fn)
		}
	}
}

func (t *tree) Text(id NodeID) string {
	n := &t.nodes[id]

	switch n.typ {
	case NodeType_String:
		return string(t.input[n.start:n.end])

	case NodeType_Sequence:
		var b strings.Builder
		for _, childID := range t.Children(id) {
			b.WriteString(t.Text(childID))
		}
		return b.String()

	case NodeType_Node:
		if child, ok := t.Child(id); ok {
			return t.Text(child)
		}
		return ""

	case NodeType_Error:
		if child, ok := t.Child(id); ok {
			return t.Text(child)
		}
		return fmt.Sprintf("error[%s]", t.Name(id))
	default:
		panic(fmt.Sprintf("Unknown node type: %T", n.typ))
	}
}

func (t *tree) Pretty(id NodeID) string {
	vi := newPrettyPrinter(t, t.input, func(input string, _ FormatToken) string {
		return input
	})
	vi.visit(id)
	return vi.output.String()
}

func (t *tree) Highlight(id NodeID) string {
	vi := newPrettyPrinter(t, t.input, func(input string, token FormatToken) string {
		return treePrinterTheme[token] + input + treePrinterTheme[FormatToken_None]
	})
	vi.visit(id)
	return vi.output.String()
}

var treePrinterTheme = map[FormatToken]string{
	FormatToken_None:    "\033[0m",          // reset
	FormatToken_Range:   "\033[1;31;5;228m", // orange
	FormatToken_Literal: "\033[1;38;5;245m", // gray
	FormatToken_Error:   "\033[1;38;5;127m", // pink
}

type prettyPrinter struct {
	input []byte
	tree  *tree
	*treePrinter[FormatToken]
}

func newPrettyPrinter(tree *tree, input []byte, format FormatFunc[FormatToken]) *prettyPrinter {
	return &prettyPrinter{
		tree:        tree,
		input:       input,
		treePrinter: newTreePrinter(format),
	}
}

func (vi *prettyPrinter) visit(id NodeID) {
	n := &vi.tree.nodes[id]

	switch n.typ {
	case NodeType_String:
		text := string(vi.input[n.start:n.end])
		escaped := strconv.Quote(text)
		vi.write(vi.format(escaped, FormatToken_Literal))
		vi.write(vi.format(fmt.Sprintf(" (%s)", vi.formatPosition(n.start, n.end)), FormatToken_Range))

	case NodeType_Sequence:
		children := vi.tree.Children(id)
		seq := fmt.Sprintf("Sequence<%d> (%s)", len(children), vi.formatPosition(n.start, n.end))
		vi.writel(vi.format(seq, FormatToken_Range))
		for i, child := range children {
			switch {
			case i == len(children)-1:
				vi.pwrite("└── ")
				vi.indent("    ")
				vi.visit(child)
				vi.unindent()
			default:
				vi.pwrite("├── ")
				vi.indent("│   ")
				vi.visit(child)
				vi.unindent()
				vi.write("\n")
			}
		}

	case NodeType_Node:
		name := vi.tree.Name(id)
		rgst := fmt.Sprintf(" (%s)", vi.formatPosition(n.start, n.end))
		vi.write(vi.format(name, FormatToken_Literal))
		vi.writel(vi.format(rgst, FormatToken_Range))
		vi.pwrite("└── ")
		vi.indent("    ")
		if child, ok := vi.tree.Child(id); ok {
			vi.visit(child)
		}
		vi.unindent()

	case NodeType_Error:
		label := vi.tree.Name(id)
		vi.write(vi.format(fmt.Sprintf("Error<%s>", label), FormatToken_Error))
		vi.write(vi.format(fmt.Sprintf(" (%s)", vi.formatPosition(n.start, n.end)), FormatToken_Range))

		if child, ok := vi.tree.Child(id); ok {
			vi.writel("")
			vi.pwrite("└── ")
			vi.indent("    ")
			vi.visit(child)
			vi.unindent()
		}
	}
}

// formatPosition formats a Range as "startLine:startCol-endLine:endCol"
func (vi *prettyPrinter) formatPosition(start, end int) string {
	startLine, startCol := vi.posToLineCol(start)
	endLine, endCol := vi.posToLineCol(end)
	if startLine == endLine && startLine == 1 {
		if startCol == endCol {
			return fmt.Sprintf("%d", startCol)
		}
		return fmt.Sprintf("%d..%d", startCol, endCol)
	}
	if startLine == endLine && startCol == endCol {
		return fmt.Sprintf("%d:%d", startLine, startCol)
	}
	return fmt.Sprintf("%d:%d..%d:%d", startLine, startCol, endLine, endCol)
}

// posToLineCol converts a byte position in the input to line and
// column numbers (both 1-based)
func (vi *prettyPrinter) posToLineCol(pos int) (line, column int) {
	line, column = 1, 1
	data := vi.input[0:pos]
	for _, ch := range data {
		if ch == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}
