package langlang

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// TreeToSExpr serializes the subtree at id to an s-expression string.
// Format: (Name child) for nodes, (Seq a b c) for sequences, "text" for strings.
func TreeToSExpr(t Tree, id NodeID) string {
	switch t.Type(id) {
	case NodeType_String:
		return strconv.Quote(t.Text(id))
	case NodeType_Node:
		name := t.Name(id)
		child, ok := t.Child(id)
		if !ok {
			return "(" + name + ")"
		}
		return "(" + name + " " + TreeToSExpr(t, child) + ")"
	case NodeType_Sequence:
		children := t.Children(id)
		parts := make([]string, 0, len(children)+1)
		parts = append(parts, "Seq")
		for _, c := range children {
			parts = append(parts, TreeToSExpr(t, c))
		}
		return "(" + strings.Join(parts, " ") + ")"
	default:
		return fmt.Sprintf("(?%s)", t.Type(id))
	}
}

// ParseSExpr parses an s-expression string and builds a tree via TreeBuilder.
// Returns the tree and the root NodeID. Format: (Name arg...) or "string".
func ParseSExpr(s string) (Tree, NodeID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, 0, fmt.Errorf("empty s-expr")
	}
	b := NewTreeBuilder()
	id, rest, err := parseSExprOne(s, b)
	if err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(rest) != "" {
		return nil, 0, fmt.Errorf("trailing input after s-expr")
	}
	return b.Tree(), id, nil
}

func parseSExprOne(s string, b *TreeBuilder) (NodeID, string, error) {
	s = strings.TrimLeft(s, " \t\n\r")
	if s == "" {
		return 0, "", fmt.Errorf("unexpected end of input")
	}
	if s[0] == '"' {
		// String literal
		end := 1
		for end < len(s) {
			if s[end] == '\\' && end+1 < len(s) {
				end += 2
				continue
			}
			if s[end] == '"' {
				end++
				break
			}
			end++
		}
		if end > len(s) {
			return 0, "", fmt.Errorf("unterminated string")
		}
		unquoted, err := strconv.Unquote(s[:end])
		if err != nil {
			return 0, "", err
		}
		return b.Str(unquoted), s[end:], nil
	}
	if s[0] != '(' {
		return 0, "", fmt.Errorf("expected '(' or '\"', got %q", s[0])
	}
	s = s[1:]
	s = strings.TrimLeft(s, " \t\n\r")
	if s == "" {
		return 0, "", fmt.Errorf("empty list")
	}
	// Read name (first atom)
	var name string
	i := 0
	for i < len(s) && !unicode.IsSpace(rune(s[i])) && s[i] != ')' && s[i] != '(' {
		i++
	}
	name = s[:i]
	s = strings.TrimLeft(s[i:], " \t\n\r")
	if name == "" {
		return 0, "", fmt.Errorf("expected name in list")
	}
	var children []NodeID
	for len(s) > 0 && s[0] != ')' {
		child, rest, err := parseSExprOne(s, b)
		if err != nil {
			return 0, "", err
		}
		children = append(children, child)
		s = strings.TrimLeft(rest, " \t\n\r")
	}
	if len(s) == 0 || s[0] != ')' {
		return 0, "", fmt.Errorf("expected ')'")
	}
	s = s[1:]

	if name == "Seq" {
		if len(children) == 0 {
			return b.Seq(), s, nil
		}
		return b.Seq(children...), s, nil
	}
	// Named node
	if len(children) == 0 {
		return b.Named(name, b.Str("")), s, nil
	}
	if len(children) == 1 {
		return b.Named(name, children[0]), s, nil
	}
	return b.Named(name, b.Seq(children...)), s, nil
}
