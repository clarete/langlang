package langlang

import "fmt"

const spacingIdentifier = "Spacing"

type whitespaceInjector struct {
	lexLevel int
}

func InjectWhitespaces(n AstNode) (AstNode, error) {
	wi := &whitespaceInjector{}
	switch node := n.(type) {
	case *GrammarNode:
		return wi.Run(node), nil
	default:
		return nil, fmt.Errorf("Expected GrammarNode, received %T", node)
	}
}

func (wi *whitespaceInjector) Run(n *GrammarNode) *GrammarNode {
	var (
		defs   = make([]*DefinitionNode, 0, len(n.Definitions))
		defMap = make(map[string]*DefinitionNode, len(n.Definitions))
	)
	for _, def := range n.Definitions {
		expr := wi.expandExpr(def.Expr, true)
		newDef := NewDefinitionNode(def.Name, expr, def.Span())
		defs = append(defs, newDef)
		defMap[def.Name] = newDef
	}
	return NewGrammarNode(n.Imports, defs, defMap, n.Span())
}

func (wi *whitespaceInjector) expandExpr(n AstNode, consumeFirst bool) AstNode {
	switch node := n.(type) {
	case *LexNode:
		wi.lexLevel++
		expr := wi.expandExpr(node.Expr, true)
		wi.lexLevel--
		return NewLexNode(expr, n.Span())

	case *SequenceNode:
		shouldConsumeSpaces := wi.lexLevel == 0 && !n.IsSyntactic()
		newItems := make([]AstNode, 0, len(node.Items))
		for i, item := range node.Items {
			idNode, isIdNode := item.(*IdentifierNode)

			isSpacingNode := isIdNode && idNode.Value == spacingIdentifier

			_, isLexNode := item.(*LexNode)

			skip := !consumeFirst && i == 0 || isLexNode || isSpacingNode

			if shouldConsumeSpaces && !skip {
				newItems = append(newItems, wsCall())
			}

			newItems = append(newItems, wi.expandExpr(item, true))
		}
		return NewSequenceNode(newItems, node.Span())

	case *ChoiceNode:
		newItems := make([]AstNode, len(node.Items))

		// No need to inject whitespace handling, we just
		// return the choice with all its child nodes
		// expanded, but the choice node itself is untouched.
		if node.IsSyntactic() {
			for i, item := range node.Items {
				newItems[i] = wi.expandExpr(item, true)
			}
			return NewChoiceNode(newItems, n.Span())
		}

		// expand expresion for each alternative within the
		// choice.  Notice that we're disabling the
		// `consumeFirst` flag here to prevent duplicating the
		// whitespace handler
		for i, item := range node.Items {
			newItems[i] = wi.expandExpr(item, false)
		}

		// Wrap the choice node in a sequence in which the
		// first element is the whitespace handling
		return NewSequenceNode([]AstNode{
			wsCall(),
			NewChoiceNode(newItems, n.Span()),
		}, node.Span())

	case *NotNode:
		return NewNotNode(wi.expandExpr(node.Expr, true), n.Span())

	case *AndNode:
		return NewAndNode(wi.expandExpr(node.Expr, true), n.Span())

	case *OptionalNode:
		return NewOptionalNode(wi.expandExpr(node.Expr, true), n.Span())

	case *ZeroOrMoreNode:
		return NewZeroOrMoreNode(wi.expandExpr(node.Expr, true), n.Span())

	case *OneOrMoreNode:
		return NewOneOrMoreNode(wi.expandExpr(node.Expr, true), n.Span())

	case *LabeledNode:
		return NewLabeledNode(node.Label, wi.expandExpr(node.Expr, true), n.Span())

	default:
		return node
	}
}

func wsCall() AstNode {
	return NewIdentifierNode(spacingIdentifier, Span{})
}
