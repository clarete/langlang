package langlang

import "fmt"

type AstNodeVisitor interface {
	VisitGrammarNode(*GrammarNode) error
	VisitImportNode(*ImportNode) error
	VisitDefinitionNode(*DefinitionNode) error
	VisitCaptureNode(*CaptureNode) error
	VisitSequenceNode(*SequenceNode) error
	VisitOneOrMoreNode(*OneOrMoreNode) error
	VisitZeroOrMoreNode(*ZeroOrMoreNode) error
	VisitOptionalNode(*OptionalNode) error
	VisitChoiceNode(*ChoiceNode) error
	VisitAndNode(*AndNode) error
	VisitNotNode(*NotNode) error
	VisitLexNode(*LexNode) error
	VisitLabeledNode(*LabeledNode) error
	VisitLiteralNode(*LiteralNode) error
	VisitClassNode(*ClassNode) error
	VisitRangeNode(*RangeNode) error
	VisitCharsetNode(*CharsetNode) error
	VisitAnyNode(*AnyNode) error
	VisitIdentifierNode(*IdentifierNode) error
	VisitErrorNode(*ErrorNode) error
}

func WalkGrammarNode(g AstNodeVisitor, n *GrammarNode) error {
	for _, item := range n.GetItems() {
		if err := item.Accept(g); err != nil {
			return err
		}
	}
	return nil
}

func WalkSequenceNode(g AstNodeVisitor, n *SequenceNode) error {
	for _, item := range n.Items {
		if err := item.Accept(g); err != nil {
			return err
		}
	}
	return nil
}

// Inspect traverses an AST in depth-first order. It calls the
// function f for each node in the tree. If f returns true, Inspect
// continues to traverse the node's children; if it returns false,
// Inspect skips the children of the current node.
//
// This is similar to Go's ast.Inspect and allows for simple traversal
// with a single type switch instead of implementing a full visitor
// pattern.
//
// Example usage:
//
//	Inspect(node, func(n AstNode) bool {
//	    if def, ok := n.(*DefinitionNode); ok {
//	        fmt.Println("Found definition:", def.Name)
//	    }
//	    return true // continue traversing
//	})
//
// This is intentionally designed to not require the exhaustiveness of
// the visitor.  In other words, if you need to write a traversal that
// looks into a single node type for example.
func Inspect(node AstNode, f func(AstNode) bool) {
	visited := make(map[AstNode]bool)
	inspect(node, f, visited)
}

func inspect(node AstNode, f func(AstNode) bool, visited map[AstNode]bool) {
	if node == nil {
		return
	}

	// Check for cycles to prevent infinite loops
	if visited[node] {
		return
	}
	visited[node] = true

	// Call the function on the current node
	if !f(node) {
		return
	}

	// Traverse children based on node type
	switch n := node.(type) {
	case *ErrorNode:
		inspect(n.Child, f, visited)

	case *ClassNode:
		for _, item := range n.Items {
			inspect(item, f, visited)
		}

	case *OptionalNode:
		inspect(n.Expr, f, visited)

	case *ZeroOrMoreNode:
		inspect(n.Expr, f, visited)

	case *OneOrMoreNode:
		inspect(n.Expr, f, visited)

	case *AndNode:
		inspect(n.Expr, f, visited)

	case *NotNode:
		inspect(n.Expr, f, visited)

	case *LexNode:
		inspect(n.Expr, f, visited)

	case *LabeledNode:
		inspect(n.Expr, f, visited)

	case *SequenceNode:
		for _, item := range n.Items {
			inspect(item, f, visited)
		}

	case *ChoiceNode:
		inspect(n.Left, f, visited)
		inspect(n.Right, f, visited)

	case *CaptureNode:
		inspect(n.Expr, f, visited)

	case *DefinitionNode:
		inspect(n.Expr, f, visited)

	case *ImportNode:
		// ImportNode has Path and Names which are LiteralNode
		// pointers We could traverse them if needed
		if n.Path != nil {
			inspect(n.Path, f, visited)
		}
		for _, name := range n.Names {
			if name != nil {
				inspect(name, f, visited)
			}
		}

	case *GrammarNode:
		for _, imp := range n.Imports {
			inspect(imp, f, visited)
		}
		for _, def := range n.Definitions {
			inspect(def, f, visited)
		}

	case *AnyNode, *LiteralNode, *IdentifierNode, *RangeNode, *CharsetNode:
		// Leaf nodes, so no children to traverse

	default:
		panic(fmt.Sprintf("Inspect is outdated, missing node %T", n))
	}
}
