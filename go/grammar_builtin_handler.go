package langlang

import (
	_ "embed"
	"fmt"
)

var (
	//go:embed builtins.peg
	builtinsText    []byte
	builtinsGrammar *GrammarNode = mustLoadBuiltins()
)

// mustLoadBuiltins will parse the grammar within `builtins.peg` and
// save the output into the global variable `builtinsGrammar`.
func mustLoadBuiltins() *GrammarNode {
	p := NewGrammarParser(builtinsText)
	p.SetGrammarFile("builtins.peg")

	parsed, err := p.Parse()
	if err != nil {
		panic(err)
	}

	node, ok := parsed.(*GrammarNode)
	if !ok {
		panic(fmt.Errorf("grammar expected, but got %#v", parsed))
	}
	return node
}

// AddBuiltins will extend the grammar received in `n` with all the
// productions found within `builtins.peg`.
func AddBuiltins(n AstNode) (*GrammarNode, error) {
	originalGrammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}

	for _, def := range builtinsGrammar.Definitions {
		originalGrammar.AddDefinition(def)
	}

	return originalGrammar, nil
}
