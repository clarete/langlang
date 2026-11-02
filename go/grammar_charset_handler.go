package langlang

import "fmt"

func AddCharsets(n AstNode) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}
	defs := make([]*DefinitionNode, 0, len(grammar.Definitions))
	for _, def := range grammar.Definitions {
		expr, err := addCharset(def.Expr)
		if err != nil {
			return nil, err
		}
		defs = append(defs, def.withExpr(expr))
	}
	return grammar.withDefinitions(defs), nil
}

func addCharset(expr AstNode) (AstNode, error) {
	switch e := expr.(type) {
	case *SequenceNode:
		// Seq([Item]) -> Item; this essentially drops
		// Sequence nodes with a single value on them.  The
		// `Prefix` rule in langlang.peg is a good example of
		// where this transformation helps.  e.g.:
		//
		//    Prefix <- ("#" / "&" / "!")? Labeled
		//
		// Gets parsed as
		//
		//    Seq(Option(Choice(Seq(Lit("#")),
		//               Choice(Seq(Lit("&")),
		//                      Seq(Lit("!"))))),
		//        Labeled)
		//
		// We want it to become
		//
		//    Seq(Option(Choice(Lit("#"),
		//                      Choice(Lit("&"),
		//                             Lit("!")))),
		//        Labeled)
		//
		// By applying this transformation, we allow the
		// `*ChoiceNode` case to merge these literals into a
		// charset.
		if len(e.Items) == 1 {
			return addCharset(e.Items[0])
		}

		// Recurse into the sequence normally
		items := make([]AstNode, len(e.Items))
		for i, item := range e.Items {
			var err error
			items[i], err = addCharset(item)
			if err != nil {
				return nil, err
			}
		}

		// Seq(Not(Lit(l0) Any)); this replaces ll that with
		// the complement of the charset `[l0]`.  Which also
		// means anything, but `l0`.
		if len(items) == 2 {
			fst, snd := items[0], items[1]
			not, fstIsNot := fst.(*NotNode)
			_, sndIsAny := snd.(*AnyNode)
			if fstIsNot && sndIsAny {
				switch ee := not.Expr.(type) {
				case *CharsetNode:
					return NewCharsetNode(ee.cs.complement(), e.SourceLocation()), nil
				case *LiteralNode:
					if len(ee.Value) == 1 && fitcs(r(ee.Value)) {
						cs := newCharsetForRune(r(ee.Value))
						return NewCharsetNode(cs.complement(), e.SourceLocation()), nil
					}
				}
			}
		}
		return NewSequenceNode(items, e.SourceLocation()), nil

	case *ClassNode:
		var classCharset *charset = nil
		for _, item := range e.Items {
			var cs *charset = nil
			switch it := item.(type) {
			case *RangeNode:
				if !fitcs(it.Left) || !fitcs(it.Right) {
					return e, nil
				}
				if it.Left > it.Right {
					return nil, fmt.Errorf("range out of bounds, did you mean `%c-%c`?", it.Right, it.Left)
				}
				cs = newCharsetForRange(it.Left, it.Right)
			case *LiteralNode:
				if !fitcs(r(it.Value)) {
					return e, nil
				}
				cs = newCharsetForRune(r(it.Value))
			}
			if classCharset == nil {
				classCharset = cs
				continue
			}
			classCharset = charsetMerge(classCharset, cs)
		}
		return NewCharsetNode(classCharset, e.SourceLocation()), nil

	case *ChoiceNode:
		lh, err := addCharset(e.Left)
		if err != nil {
			return nil, err
		}
		rh, err := addCharset(e.Right)
		if err != nil {
			return nil, err
		}
		sl, slOk := lh.(*CharsetNode)
		sr, srOk := rh.(*CharsetNode)
		if slOk && srOk {
			cs := charsetMerge(sl.cs, sr.cs)
			return NewCharsetNode(cs, e.SourceLocation()), nil
		}

		return NewChoiceNode(lh, rh, e.SourceLocation()), nil

	case *LiteralNode:
		if len(e.Value) == 1 && fitcs(r(e.Value)) {
			cs := newCharsetForRune(r(e.Value))
			return NewCharsetNode(cs, e.SourceLocation()), nil
		}

	case *OptionalNode:
		child, err := addCharset(e.Expr)
		return NewOptionalNode(child, e.SourceLocation()), err

	case *ZeroOrMoreNode:
		child, err := addCharset(e.Expr)
		return NewZeroOrMoreNode(child, e.SourceLocation()), err

	case *OneOrMoreNode:
		child, err := addCharset(e.Expr)
		return NewOneOrMoreNode(child, e.SourceLocation()), err

	case *LexNode:
		child, err := addCharset(e.Expr)
		return NewLexNode(child, e.SourceLocation()), err

	case *LabeledNode:
		child, err := addCharset(e.Expr)
		return NewLabeledNode(e.Label, child, e.SourceLocation()), err

	case *NotNode:
		child, err := addCharset(e.Expr)
		return NewNotNode(child, e.SourceLocation()), err

	case *AndNode:
		child, err := addCharset(e.Expr)
		return NewAndNode(child, e.SourceLocation()), err

	case *CaptureNode:
		child, err := addCharset(e.Expr)
		return NewCaptureNode(e.Name, child, e.SourceLocation()), err
	}
	return expr, nil
}
