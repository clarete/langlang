package langlang

import "fmt"

func AddCharsets(n AstNode) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}
	for _, def := range grammar.Definitions {
		def.Expr = addCharset(def.Expr)
	}
	return grammar, nil
}

func addCharset(expr AstNode) AstNode {
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
			e.Items[i] = addCharset(item)
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
					return NewCharsetNode(ee.cs.complement(), e.Range())
				case *LiteralNode:
					if len(ee.Value) == 1 && fitcs(r(ee.Value)) {
						cs := newCharsetForRune(r(ee.Value))
						return NewCharsetNode(cs.complement(), e.Range())
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
					return e
				}
				cs = newCharsetForRange(it.Left, it.Right)
			case *LiteralNode:
				if !fitcs(r(it.Value)) {
					return e
				}
				cs = newCharsetForRune(r(it.Value))
			}
			if classCharset == nil {
				classCharset = cs
				continue
			}
			classCharset = charsetMerge(classCharset, cs)
		}
		return NewCharsetNode(classCharset, e.Range())

	case *ChoiceNode:
		lh := addCharset(e.Left)
		rh := addCharset(e.Right)
		sl, slOk := lh.(*CharsetNode)
		sr, srOk := rh.(*CharsetNode)
		if slOk && srOk {
			cs := charsetMerge(sl.cs, sr.cs)
			return NewCharsetNode(cs, e.Range())
		}

		e.Left = lh
		e.Right = rh

	case *LiteralNode:
		if len(e.Value) == 1 && fitcs(r(e.Value)) {
			cs := newCharsetForRune(r(e.Value))
			return NewCharsetNode(cs, e.Range())
		}

	case *OptionalNode:
		e.Expr = addCharset(e.Expr)

	case *ZeroOrMoreNode:
		e.Expr = addCharset(e.Expr)

	case *OneOrMoreNode:
		e.Expr = addCharset(e.Expr)

	case *LexNode:
		e.Expr = addCharset(e.Expr)

	case *LabeledNode:
		e.Expr = addCharset(e.Expr)

	case *NotNode:
		e.Expr = addCharset(e.Expr)

	case *AndNode:
		e.Expr = addCharset(e.Expr)

	case *CaptureNode:
		e.Expr = addCharset(e.Expr)
	}
	return expr
}
