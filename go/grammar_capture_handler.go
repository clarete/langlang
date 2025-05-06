package langlang

import "fmt"

func AddCaptures(n AstNode) (*GrammarNode, error) {
	grammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("Grammar expected, but got %#v", n)
	}

	for _, def := range grammar.Definitions {
		def.Expr = NewCaptureNode(def.Name, def.Expr, def.Span())
	}

	return grammar, nil
}
