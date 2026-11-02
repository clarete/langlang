package langlang

import "fmt"

func AddCaptures(n AstNode, cfg *Config) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}
	var (
		rg       = newRuleGraph(grammar)
		recovery = rg.LabelTargets()
		spacing  = rg.SpacingClosure()
		enabled  = cfg.GetBool("grammar.capture_spaces")
		defs     = make([]*DefinitionNode, 0, len(grammar.Definitions))
	)
	for _, def := range grammar.Definitions {
		if _, skip := spacing[def.Name]; skip && !enabled {
			defs = append(defs, def)
			continue
		}

		if _, isRecovery := recovery[def.Name]; !isRecovery && isZeroWidth(def.Expr) {
			defs = append(defs, def)
			continue
		}

		expr := def.Expr
		if !isSyntactic(def, false) {
			expr = addUnamedCaptures(expr)
		}
		defs = append(defs, def.withExpr(NewCaptureNode(def.Name, expr, def.SourceLocation())))
	}
	return grammar.withDefinitions(defs), nil
}

func addUnamedCaptures(expr AstNode) AstNode {
	switch e := expr.(type) {
	case *SequenceNode:
		items := make([]AstNode, len(e.Items))
		for i, item := range e.Items {
			items[i] = addUnamedCaptures(item)
		}
		return NewSequenceNode(items, e.SourceLocation())

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
			choice := NewChoiceNode(cl.Expr, cr.Expr, e.SourceLocation())
			return NewCaptureNode("", choice, e.SourceLocation())
		}
		return NewChoiceNode(l, r, e.SourceLocation())

	case *OptionalNode:
		return NewOptionalNode(addUnamedCaptures(e.Expr), e.SourceLocation())

	case *ZeroOrMoreNode:
		i := addUnamedCaptures(e.Expr)

		// Rewrite: Star(Cap(Syntactic())) -> Cap(Star(Syntactic()))
		if cap, isCap := i.(*CaptureNode); isCap && isSyntactic(cap.Expr, false) {
			body := NewZeroOrMoreNode(cap.Expr, e.SourceLocation())
			return NewCaptureNode(cap.Name, body, cap.SourceLocation())
		}

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
				body := NewSequenceNode([]AstNode{seq.Items[0], second.Expr}, seq.SourceLocation())
				star := NewZeroOrMoreNode(body, e.SourceLocation())
				return NewCaptureNode("", star, e.SourceLocation())
			}
		}
		return NewZeroOrMoreNode(i, e.SourceLocation())

	case *OneOrMoreNode:
		return NewOneOrMoreNode(addUnamedCaptures(e.Expr), e.SourceLocation())

	case *LexNode:
		return NewLexNode(addUnamedCaptures(e.Expr), e.SourceLocation())

	case *LabeledNode:
		return NewLabeledNode(e.Label, addUnamedCaptures(e.Expr), e.SourceLocation())

	case *NotNode, *AndNode:
		// predicates don't move the cursor, thus should never
		// anything to be captured so skip them altogether

	default:
		_, isCap := expr.(*CaptureNode)
		if isSyntactic(expr, false) && !isCap {
			return NewCaptureNode("", e, e.SourceLocation())
		}
	}
	return expr
}
