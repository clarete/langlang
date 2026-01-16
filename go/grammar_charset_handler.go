package langlang

import "fmt"

func AddCharsets(n AstNode) (*GrammarNode, error) {
	var err error
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}
	for _, def := range grammar.Definitions {
		def.Expr, err = addCharset(def.Expr)
		if err != nil {
			return nil, err
		}
	}
	return grammar, nil
}

func addCharset(expr AstNode) (AstNode, error) {
	var err error
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
		for i, item := range e.Items {
			e.Items[i], err = addCharset(item)
			if err != nil {
				return nil, err
			}
		}

		// Seq(Not(Lit(l0) Any)); this replaces ll that with
		// the complement of the charset `[l0]`.  Which also
		// means anything, but `l0`.
		if len(e.Items) == 2 {
			fst, snd := e.Items[0], e.Items[1]
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

		e.Left = lh
		e.Right = rh

	case *LiteralNode:
		if len(e.Value) == 1 && fitcs(r(e.Value)) {
			cs := newCharsetForRune(r(e.Value))
			return NewCharsetNode(cs, e.SourceLocation()), nil
		}

	case *OptionalNode:
		e.Expr, err = addCharset(e.Expr)

	case *ZeroOrMoreNode:
		e.Expr, err = addCharset(e.Expr)

	case *OneOrMoreNode:
		e.Expr, err = addCharset(e.Expr)

	case *LexNode:
		e.Expr, err = addCharset(e.Expr)

	case *LabeledNode:
		e.Expr, err = addCharset(e.Expr)

	case *NotNode:
		e.Expr, err = addCharset(e.Expr)

	case *AndNode:
		e.Expr, err = addCharset(e.Expr)

	case *CaptureNode:
		e.Expr, err = addCharset(e.Expr)
	}
	if err != nil {
		return nil, err
	}
	return expr, nil
}
