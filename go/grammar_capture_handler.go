package langlang

import "fmt"

// Keep in sync with `builtins.peg` or find a better way to track them
var skipAddingCaptures = map[string]struct{}{
	"Spacing": {},
	"Space":   {},
	"EOF":     {},
	"EOL":     {},
}

func AddCaptures(n AstNode, cfg *Config) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}
	// TODO: share these with `grammar_whitespace_handler`
	var (
		spDeps       = newSortedDeps()
		spDef, hasSp = grammar.DefsByName[spacingIdentifier]
	)
	if hasSp {
		spDeps.names = append(spDeps.names, spacingIdentifier)
		findDefinitionDeps(grammar, spDef, spDeps)
		for _, name := range spDeps.names {
			skipAddingCaptures[name] = struct{}{}
		}
	}
	for _, def := range grammar.Definitions {
		if _, skip := skipAddingCaptures[def.Name]; skip && !cfg.GetBool("grammar.capture_spaces") {
			continue
		}

		expr := def.Expr
		if !isSyntactic(def, false) {
			expr = addUnamedCaptures(expr)
		}
		def.Expr = NewCaptureNode(def.Name, expr, def.Range())
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
			expr = NewCaptureNode("", e, e.Range())
		} else {
			e.Left = l
			e.Right = r
		}

	case *OptionalNode:
		e.Expr = addUnamedCaptures(e.Expr)

	case *ZeroOrMoreNode:
		i := addUnamedCaptures(e.Expr)

		// Rewrite: Star(Cap(Syntactic())) -> Cap(Star(Syntactic()))
		if cap, isCap := i.(*CaptureNode); isCap && isSyntactic(cap.Expr, false) {
			e.Expr = cap.Expr
			cap.Expr = e
			return cap
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
				seq.Items[1] = second.Expr
				expr = NewCaptureNode("", e, e.Range())
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
		if isSyntactic(expr, false) && !isCap {
			return NewCaptureNode("", e, e.Range())
		}
	}
	return expr
}
