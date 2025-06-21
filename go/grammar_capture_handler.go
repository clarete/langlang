package langlang

import "fmt"

func AddCaptures(n AstNode) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("Grammar expected, but got %#v", n)
	}

	for _, def := range grammar.Definitions {
		expr := def.Expr
		if !def.IsSyntactic() {
			expr = addUnamedCaptures(expr)
		}
		def.Expr = NewCaptureNode(def.Name, expr, def.Span())
	}

	return grammar, nil
}

func addUnamedCaptures(expr AstNode) AstNode {
	switch e := expr.(type) {
	case *SequenceNode:
		for i, item := range e.Items {
			e.Items[i] = addUnamedCaptures(item)
		}

	case *ChoiceNode:
		e.Left = addUnamedCaptures(e.Left)
		e.Right = addUnamedCaptures(e.Right)

	case *OptionalNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *ZeroOrMoreNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *OneOrMoreNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *LexNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *LabeledNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *NotNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *AndNode:
		e.Expr = addUnamedCaptures(e.Expr)

	default:
		if expr.IsSyntactic() {
			return NewCaptureNode("", e, e.Span())
		}
	}
	return expr
}
