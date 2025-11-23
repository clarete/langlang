package diagram

import (
	_ "embed"

	"github.com/clarete/langlang/go"
)

func fromBytes(input []byte) (diagram, error) {
	cst, _, err := diagramParser.Match(input)
	if err != nil {
		return nil, err
	}

	v := &cstVisitor{input: input, stack: []diagram{}}

	if err := cst.Accept(v); err != nil {
		return nil, err
	}

	return v.pop(), nil
}

var (
	//go:embed diagram_parser.peg
	diagramParserGrammar []byte
	diagramParser        = loadParser()
)

func loadParser() langlang.Matcher {
	cfg := langlang.NewConfig()
	cfg.SetBool("grammar.capture_spaces", false)
	matcher, err := langlang.MatcherFromBytes(diagramParserGrammar, cfg)
	if err != nil {
		panic(err.Error())
	}
	return matcher
}

type cstVisitor struct {
	input []byte
	stack []diagram
}

func (vi *cstVisitor) VisitNode(n *langlang.Node) error {
	switch n.Name {
	case "Diagram":
		return n.Expr.Accept(vi)

	case "Terminal":
		label := n.Expr.String(vi.input)
		vi.push(newterm(label[1 : len(label)-1]))
		return nil

	case "NonTerminal":
		seq := n.Expr.(*langlang.Sequence)
		item := seq.Items[1]
		label := item.String(vi.input)
		vi.push(newnonterm(label))
		return nil

	case "Sequence":
		seq := n.Expr.(*langlang.Sequence)
		items := seq.Items[1 : len(seq.Items)-1]
		if len(items) == 0 {
			vi.push(newempty())
			return nil
		}
		out := []diagram{}
		for _, item := range items {
			if err := item.Accept(vi); err != nil {
				return err
			}

			if d := vi.pop(); d != nil {
				out = append(out, d)
			}
		}
		vi.push(newseq(out))

	case "Stack":
		seq := n.Expr.(*langlang.Sequence)
		items := seq.Items[1 : len(seq.Items)-1]
		polarity := newpolarity(items[0].String(vi.input))
		if err := items[1].Accept(vi); err != nil {
			return err
		}
		top := vi.pop()
		if err := items[2].Accept(vi); err != nil {
			return err
		}
		bottom := vi.pop()
		vi.push(newstack(polarity, top, bottom))
	}
	return nil
}

func (vi *cstVisitor) VisitSequence(n *langlang.Sequence) error {
	for _, item := range n.Items {
		if err := item.Accept(vi); err != nil {
			return err
		}
	}
	return nil
}

func (vi *cstVisitor) VisitString(n *langlang.String) error { return nil }

func (vi *cstVisitor) VisitError(n *langlang.Error) error { return nil }

func (vi *cstVisitor) push(i diagram) {
	vi.stack = append(vi.stack, i)
}

func (vi *cstVisitor) pop() diagram {
	idx := len(vi.stack) - 1
	top := vi.stack[idx]
	vi.stack = vi.stack[:idx]
	return top
}
