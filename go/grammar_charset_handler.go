package langlang

import "fmt"

func AddCharsets(n AstNode) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("Grammar expected, but got %#v", n)
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

	case *ClassNode:
		var classCharset *charset = nil
		for _, item := range e.Items {
			var cs *charset
			switch it := item.(type) {
			case *RangeNode:
				cs = newCharsetForRange(it.Left, it.Right)
			case *LiteralNode:
				cs = newCharsetFromString(it.Value)
			}
			if classCharset == nil {
				classCharset = cs
			} else {
				classCharset = charsetMerge(classCharset, cs)
			}
		}
		expr = NewCharsetNode(classCharset, e.Span())

	case *ChoiceNode:
		l := addCharset(e.Left)
		r := addCharset(e.Right)

		sl, slOk := l.(*CharsetNode)
		sr, srOk := r.(*CharsetNode)

		ll, llOk := l.(*LiteralNode)
		lr, lrOk := r.(*LiteralNode)

		switch {
		case slOk && srOk:
			// two charsets
			cs := charsetMerge(sl.cs, sr.cs)
			expr = NewCharsetNode(cs, e.Span())

		case llOk && lrOk && len(ll.Value) == 1 && len(lr.Value) == 1:
			// two literals
			csl := newCharsetFromString(ll.Value)
			csr := newCharsetFromString(lr.Value)
			cs := charsetMerge(csl, csr)
			expr = NewCharsetNode(cs, e.Span())

		case slOk && lrOk && len(lr.Value) == 1:
			// charset / literal
			csr := newCharsetFromString(lr.Value)
			cs := charsetMerge(sl.cs, csr)
			expr = NewCharsetNode(cs, e.Span())

		case srOk && llOk && len(ll.Value) == 1:
			// literal / charset
			csl := newCharsetFromString(ll.Value)
			cs := charsetMerge(sr.cs, csl)
			expr = NewCharsetNode(cs, e.Span())

		default:
			e.Left = l
			e.Right = r
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

	}
	return expr
}
