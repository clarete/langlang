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

	case *PrecedenceNode:
		return isSyntactic(e.Expr, isCapture)

	case *GrammarNode:
		return false

	case *ImportNode:
		return false

	default:
		panic(fmt.Sprintf("isSyntactic: unknown node type %T", e))
	}
}

// isZeroWidth decides if the node is unambiguously going to not
// consume any chars from the input.  This is useful because the VM
// will discard empty captures anyway, we might as well get rid of
// them during compilation and make the program a bit more efficient.
//
// Labeled expressions don't get deeply checked, it's just stated as
// false and the consequence is adding captures around a node that
// might be dropped by the VM.  Also recovery expressions might need
// to be captured anyway when wrapped in error nodes.
func isZeroWidth(expr AstNode) bool {
	switch e := expr.(type) {
	case *AndNode, *NotNode:
		return true
	case *LiteralNode:
		return len(e.Value) == 0
	case *SequenceNode:
		for _, item := range e.Items {
			if !isZeroWidth(item) {
				return false
			}
		}
		return true
	case *ChoiceNode:
		return isZeroWidth(e.Left) && isZeroWidth(e.Right)
	case *OptionalNode:
		return isZeroWidth(e.Expr)
	case *ZeroOrMoreNode:
		return isZeroWidth(e.Expr)
	case *OneOrMoreNode:
		return isZeroWidth(e.Expr)
	case *LexNode:
		return isZeroWidth(e.Expr)
	case *CaptureNode:
		return isZeroWidth(e.Expr)
	case *DefinitionNode:
		return isZeroWidth(e.Expr)
	default:
		return false
	}
}
