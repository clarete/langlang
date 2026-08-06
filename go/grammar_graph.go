package langlang

type RefKind uint8

const (
	// RefKind_Call is a rule invocation: the `B` in `A <- B`.
	RefKind_Call RefKind = iota
	// RefKind_Label is a label annotation: the `L` in `e^L`.  It
	// also names a recovery rule when a definition called `L`
	// exists.
	RefKind_Label
)

type RuleRef struct {
	Name string
	Kind RefKind
	Loc  SourceLocation
}

type RuleGraph struct {
	grammar *GrammarNode
	refs    map[string][]RuleRef
}

type Caller struct {
	// From carries the name of the definition in which the
	// reference appeared
	From string
	Ref  RuleRef
}

func refs(node AstNode) []RuleRef {
	var out []RuleRef
	Inspect(node, func(n AstNode) bool {
		loc := n.SourceLocation()
		switch n := n.(type) {
		case *IdentifierNode:
			out = append(out, RuleRef{Name: n.Value, Kind: RefKind_Call, Loc: loc})
		case *LabeledNode:
			out = append(out, RuleRef{Name: n.Label, Kind: RefKind_Label, Loc: loc})
		}
		return true
	})
	return out
}

func newRuleGraph(g *GrammarNode) *RuleGraph {
	rg := &RuleGraph{
		grammar: g,
		refs:    make(map[string][]RuleRef, len(g.Definitions)),
	}
	for _, def := range g.Definitions {
		rg.refs[def.Name] = refs(def.Expr)
	}
	return rg
}

func (rg *RuleGraph) Grammar() *GrammarNode {
	return rg.grammar
}

func (rg *RuleGraph) Closure(start string) []string {
	var (
		out  []string
		walk func(name string)
		seen = map[string]struct{}{}
	)
	walk = func(name string) {
		for _, ref := range rg.refs[name] {
			if _, ok := seen[ref.Name]; ok {
				continue
			}
			if _, defined := rg.grammar.DefsByName[ref.Name]; !defined {
				// UndefinedReferencesQuery will take care of it
				continue
			}
			seen[ref.Name] = struct{}{}
			out = append(out, ref.Name)
			walk(ref.Name)
		}
	}
	walk(start)
	return out
}

func (rg *RuleGraph) SpacingClosure() map[string]struct{} {
	out := map[string]struct{}{}

	if _, exists := rg.grammar.DefsByName[spacingIdentifier]; !exists {
		return out
	}
	out[spacingIdentifier] = struct{}{}

	for _, name := range rg.Closure(spacingIdentifier) {
		out[name] = struct{}{}
	}
	return out
}

func (rg *RuleGraph) Callers() map[string][]Caller {
	out := map[string][]Caller{}
	for _, def := range rg.grammar.Definitions {
		for _, ref := range rg.refs[def.Name] {
			out[ref.Name] = append(out[ref.Name], Caller{
				From: def.Name,
				Ref:  ref,
			})
		}
	}
	return out
}

func (rg *RuleGraph) LabelTargets() map[string]struct{} {
	out := map[string]struct{}{}
	for _, refs := range rg.refs {
		for _, ref := range refs {
			if ref.Kind == RefKind_Label {
				out[ref.Name] = struct{}{}
			}
		}
	}
	return out
}
