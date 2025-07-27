package langlang

import "fmt"

func isSyntactic(expr AstNode, isCapture bool) bool {
	switch e := expr.(type) {

	case *LiteralNode, *CharsetNode, *ClassNode, *RangeNode, *AnyNode:
		// Terminals
		return true

	case *IdentifierNode:
		// Non Terminals
		return false

	case *LabeledNode:
		// For automatic spacing handling, labeled nodes are
		// effectively syntactic.  But for captures, they
		// can't be considered syntactic because they will be
		// translated into `opThrow`, which is essentially an
		// `opCall` for labels with recovery expressions.
		if isCapture {
			return isSyntactic(e.Expr, isCapture)
		}
		return false

	case *OneOrMoreNode:
		return isSyntactic(e.Expr, isCapture)

	case *ZeroOrMoreNode:
		return isSyntactic(e.Expr, isCapture)

	case *OptionalNode:
		return isSyntactic(e.Expr, isCapture)

	case *LexNode:
		return isSyntactic(e.Expr, isCapture)

	case *CaptureNode:
		return isSyntactic(e.Expr, isCapture)

	case *DefinitionNode:
		return isSyntactic(e.Expr, isCapture)

	case *ChoiceNode:
		return isSyntactic(e.Left, isCapture) && isSyntactic(e.Right, isCapture)

	case *SequenceNode:
		for _, expr := range e.Items {
			if !isSyntactic(expr, isCapture) {
				return false
			}
		}
		return true

	case *AndNode:
		return true

	case *NotNode:
		return true

	case *GrammarNode:
		return false

	case *ImportNode:
		return false

	default:
		panic(fmt.Sprintf("isSyntactic: unknown node type %T", e))
	}
}
