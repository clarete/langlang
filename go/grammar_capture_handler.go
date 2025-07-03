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
		l := addUnamedCaptures(e.Left)
		r := addUnamedCaptures(e.Right)

		cl, clOk := l.(*CaptureNode)
		cr, crOk := r.(*CaptureNode)

		// if both sides of the are wrapped with an unamed
		// capture, we wrap the original choice node in a
		// capture node instead of adding one capture per
		// choice branch.
		if clOk && cl.Name == "" && crOk && cr.Name == "" {
			expr = NewCaptureNode("", e, e.Span())
		} else {
			e.Left = l
			e.Right = r
		}

	case *OptionalNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *ZeroOrMoreNode:
		i := addUnamedCaptures(e.Expr)

		// Rewrite: (!x Cap(y))* -> Cap((!x y)*)
		//
		// This will generate way less capture frames on the
		// stack.  The `Comment` rule in `langlang.peg` is an
		// example of where this is useful:
		//
		//    Comment <- '//' (!EOL .)* EOL
		//
		// Without this rewrite, that expression will be
		// rewritten as:
		//
		//    Comment <- Cap('//') (!EOL Cap(.))* EOL
		//
		// And will produce a new node for every `.` matched.
		// Whereas with this rewrite, it becomes this instead:
		//
		//    Comment <- Cap('//') Cap((!EOL .)*) EOL
		//
		// Which generates a single capture frame for all the
		// characters within the comment.
		if seq, ok := i.(*SequenceNode); ok && len(seq.Items) == 2 {
			_, firstIsNot := seq.Items[0].(*NotNode)
			second, secondIsCapture := seq.Items[1].(*CaptureNode)
			if firstIsNot && secondIsCapture && second.Name == "" {
				// unwrap item within capture, and wrap the whole expr
				seq.Items[1] = second.Expr
				expr = NewCaptureNode("", e, e.Span())
				return expr
			}
		}
		e.Expr = addUnamedCaptures(e.Expr)

	case *OneOrMoreNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *LexNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *LabeledNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *NotNode, *AndNode:
		// predicates don't move the cursor, thus should never
		// anything to be captured so skip them altogether

	default:
		_, isCap := expr.(*CaptureNode)
		if expr.IsSyntactic() != isCap {
			return NewCaptureNode("", e, e.Span())
		}
	}
	return expr
}
