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
		defs    = make([]*DefinitionNode, 0, len(n.Definitions))
		spacing = newRuleGraph(n).SpacingClosure()
	)
	for _, def := range n.Definitions {
		// Spacing and its dependencies keep their own spacing policy.
		if _, skip := spacing[def.Name]; skip {
			defs = append(defs, def)
			continue
		}
		defs = append(defs, def.withExpr(wi.expandExpr(def.Expr, true)))
	}
	return n.withDefinitions(defs)
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
				// Insert Spacing *within* the OptionalNode, that way spacing gets rolled
				// back when the optional expression fails.  This is also consistent with
				// what we do to both {Zero,One}OrMore.
				if opt, ok := item.(*OptionalNode); ok {
					expr := wi.expandExpr(opt.Expr, false)
					seq := NewSequenceNode([]AstNode{wsCall(), expr}, opt.SourceLocation())
					newItems = append(newItems, NewOptionalNode(seq, opt.SourceLocation()))
					continue
				}
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
			return NewChoiceNode(
				wi.expandExpr(node.Left, true),
				wi.expandExpr(node.Right, true),
				n.SourceLocation(),
			)
		}

		// expand expresion for each alternative within the
		// choice.  Notice that we're disabling the
		// `consumeFirst` flag here to prevent duplicating the
		// whitespace handler
		return NewChoiceNode(
			wi.expandExpr(node.Left, false),
			wi.expandExpr(node.Right, false),
			n.SourceLocation(),
		)

	case *NotNode:
		return NewNotNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *AndNode:
		return NewAndNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *OptionalNode:
		return NewOptionalNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *ZeroOrMoreNode:
		shouldConsumeSpaces := wi.lexLevel == 0 && !isSyntactic(n, true)
		if shouldConsumeSpaces {
			expr := wi.expandExpr(node.Expr, false)
			seq := NewSequenceNode([]AstNode{wsCall(), expr}, node.SourceLocation())
			return NewZeroOrMoreNode(seq, n.SourceLocation())
		}
		return NewZeroOrMoreNode(wi.expandExpr(node.Expr, true), n.SourceLocation())

	case *OneOrMoreNode:
		shouldConsumeSpaces := wi.lexLevel == 0 && !isSyntactic(n, true)
		if shouldConsumeSpaces {
			expr := wi.expandExpr(node.Expr, false)
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
