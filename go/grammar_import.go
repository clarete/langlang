package langlang

const unknownFileID FileID = -1

func NewSourceLocation(f FileID, s Span) SourceLocation {
	return SourceLocation{FileID: f, Span: s}
}

// sortedDeps tracks definition dependencies in insertion order.
// Used by import resolution and grammar transformations.
type sortedDeps struct {
	names []string
	nodes map[string]*DefinitionNode
}

func newSortedDeps() *sortedDeps {
	return &sortedDeps{names: []string{}, nodes: map[string]*DefinitionNode{}}
}

// findDefinitionDeps traverses the definition `node` and finds all
// identifiers within it.  If the identifier hasn't been seen yet, it
// will add it to the dependency list, and traverse into the
// definition that points into that identifier.
func findDefinitionDeps(g *GrammarNode, node AstNode, deps *sortedDeps) error {
	switch n := node.(type) {
	case *DefinitionNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *IdentifierNode:
		// Let's not recurse if this dep has been seen already
		if _, ok := deps.nodes[n.Value]; ok {
			return nil
		}

		// save definition as a dependency and recurse into it
		def, ok := g.DefsByName[n.Value]
		if !ok {
			// Skip undefined references as they will be caught by semantic
			// analysis (through UndefinedReferencesQuery) with proper
			// source location info
			return nil
		}
		deps.nodes[n.Value] = def
		deps.names = append(deps.names, n.Value)
		return findDefinitionDeps(g, def.Expr, deps)
	case *SequenceNode:
		for _, item := range n.Items {
			if err := findDefinitionDeps(g, item, deps); err != nil {
				return err
			}
		}
		return nil
	case *ChoiceNode:
		if err := findDefinitionDeps(g, n.Left, deps); err != nil {
			return err
		}
		if err := findDefinitionDeps(g, n.Right, deps); err != nil {
			return err
		}
		return nil
	case *OptionalNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *ZeroOrMoreNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *OneOrMoreNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *AndNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *NotNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *LexNode:
		return findDefinitionDeps(g, n.Expr, deps)
	case *LabeledNode:
		// save definition as a dependency and recurse into it
		if def, ok := g.DefsByName[n.Label]; ok {
			deps.nodes[n.Label] = def
			deps.names = append(deps.names, n.Label)
			if err := findDefinitionDeps(g, def.Expr, deps); err != nil {
				return err
			}
		}
		return findDefinitionDeps(g, n.Expr, deps)
	default:
		return nil
	}
}
