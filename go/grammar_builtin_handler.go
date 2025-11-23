package langlang

import (
	_ "embed"
	"fmt"
)

// AddBuiltins will extend the grammar received in `n` with all the
// productions found within `builtins.peg`.
func AddBuiltins(n AstNode) (*GrammarNode, error) {
	originalGrammar, ok := n.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", n)
	}

	// Re-parse builtins each time to avoid sharing mutable state across
	// different grammar instances. This prevents test interference where
	// modifications to AST nodes in one test affect other tests.
	freshBuiltins := mustLoadBuiltins()
	for _, def := range freshBuiltins.Definitions {
		originalGrammar.AddDefinition(def)
	}

	return originalGrammar, nil
}

//go:embed builtins.peg
var builtinsText []byte

// mustLoadBuiltins will parse the grammar within `builtins.peg`
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
