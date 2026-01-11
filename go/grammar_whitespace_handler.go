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
		return nil, fmt.Errorf("expected GrammarNode, received %T", node)
	}
}

func (wi *whitespaceInjector) Run(n *GrammarNode) *GrammarNode {
	var (
		defs         = make([]*DefinitionNode, 0, len(n.Definitions))
		defMap       = make(map[string]*DefinitionNode, len(n.Definitions))
		spDeps       = newSortedDeps()
		spDef, hasSp = n.DefsByName[spacingIdentifier]
	)
	if hasSp {
		spDeps.names = append(spDeps.names, spacingIdentifier)
		findDefinitionDeps(n, spDef, spDeps)
	}
outer:
	for _, def := range n.Definitions {
		// Avoid injecting `Spacing` within the `Spacing` rule
		// and its dependencies
		for _, dep := range spDeps.names {
			if def.Name == dep {
				defs = append(defs, def)
				defMap[def.Name] = def
				continue outer
			}
		}
		expr := wi.expandExpr(def.Expr, true)
		newDef := NewDefinitionNode(def.Name, expr, def.SourceLocation())
		defs = append(defs, newDef)
		defMap[def.Name] = newDef
	}
	return NewGrammarNode(n.Imports, defs, defMap, n.SourceLocation())
}

func (wi *whitespaceInjector) expandExpr(n AstNode, consumeFirst bool) AstNode {
	switch node := n.(type) {
	case *LexNode:
		wi.lexLevel++
		expr := wi.expandExpr(node.Expr, true)
		wi.lexLevel--
		return NewLexNode(expr, n.SourceLocation())

	case *SequenceNode:
		shouldConsumeSpaces := wi.lexLevel == 0 && !isSyntactic(n, true)
		newItems := make([]AstNode, 0, len(node.Items))
		for i, item := range node.Items {
			idNode, isIdNode := item.(*IdentifierNode)

			isSpacingNode := isIdNode && idNode.Value == spacingIdentifier

			_, isLexNode := item.(*LexNode)
			_, isSeqNode := item.(*SequenceNode)
			_, isZeroOrMore := item.(*ZeroOrMoreNode)
			_, isOneOrMore := item.(*OneOrMoreNode)

			skip := !consumeFirst && i == 0 || (isLexNode ||
				isSeqNode ||
				isSpacingNode ||
				isZeroOrMore ||
				isOneOrMore)

			if shouldConsumeSpaces && !skip {
				newItems = append(newItems, wsCall())
			}
			newItems = append(newItems, wi.expandExpr(item, true))
		}
		return NewSequenceNode(newItems, node.SourceLocation())

	case *ChoiceNode:
		// No need to inject whitespace handling, we just
		// return the choice with all its child nodes
		// expanded, but the choice node itself is untouched.
		if isSyntactic(node, true) {
			node.Left = wi.expandExpr(node.Left, true)
			node.Right = wi.expandExpr(node.Right, true)
			return node
		}

		// expand expresion for each alternative within the
		// choice.  Notice that we're disabling the
		// `consumeFirst` flag here to prevent duplicating the
		// whitespace handler
		node.Left = wi.expandExpr(node.Left, false)
		node.Right = wi.expandExpr(node.Right, false)
		return node

	case *NotNode:
		return NewNotNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *AndNode:
		return NewAndNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *OptionalNode:
		return NewOptionalNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *ZeroOrMoreNode:
		shouldConsumeSpaces := wi.lexLevel == 0 && !isSyntactic(n, true)
		if shouldConsumeSpaces {
			expr := wi.expandExpr(node.Expr, true)
			seq := NewSequenceNode([]AstNode{wsCall(), expr}, node.SourceLocation())
			return NewZeroOrMoreNode(seq, n.SourceLocation())
		}
		return NewZeroOrMoreNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *OneOrMoreNode:
		shouldConsumeSpaces := wi.lexLevel == 0 && !isSyntactic(n, true)
		if shouldConsumeSpaces {
			expr := wi.expandExpr(node.Expr, true)
			seq := NewSequenceNode([]AstNode{wsCall(), expr}, node.SourceLocation())
			return NewOneOrMoreNode(seq, n.SourceLocation())
		}
		return NewOneOrMoreNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *LabeledNode:
		return NewLabeledNode(node.Label, wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *CaptureNode:
		return NewCaptureNode(node.Name, wi.expandExpr(node.Expr, true), n.SourceLocation())

	default:
		return node
	}
}

func wsCall() AstNode {
	return NewIdentifierNode(spacingIdentifier, SourceLocation{})
}
