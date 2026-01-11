package langlang

import "fmt"

const unknownFileID FileID = -1

func NewSourceLocation(f FileID, s Span) SourceLocation {
	return SourceLocation{FileID: f, Span: s}
}

type ImportResolver struct {
	loader ImportLoader
	nextID FileID
	intern map[string]FileID
	paths  map[FileID]string
}

func NewImportResolver(loader ImportLoader) *ImportResolver {
	return &ImportResolver{
		loader: loader,
		nextID: unknownFileID,
		intern: map[string]FileID{},
		paths:  map[FileID]string{},
	}
}

func (r *ImportResolver) Resolve(source string, cfg *Config) (AstNode, error) {
	f, err := r.resolve(source, source)
	if err != nil {
		return nil, err
	}
	f.Grammar.SourceFiles = r.grammarFiles()
	return GrammarTransformations(f.Grammar, cfg)
}

func (r *ImportResolver) MatcherFor(entry string, cfg *Config) (Matcher, error) {
	ast, err := r.Resolve(entry, cfg)
	if err != nil {
		return nil, err
	}
	asm, err := Compile(ast, cfg)
	if err != nil {
		return nil, err
	}
	code := Encode(asm, cfg)
	vm := NewVirtualMachine(code)
	vm.SetShowFails(cfg.GetBool("vm.show_fails"))
	return vm, nil
}

func (r *ImportResolver) GetPath(fileID FileID) string {
	return r.paths[fileID]
}

func (r *ImportResolver) grammarFiles() []string {
	if r.nextID < 0 {
		return nil
	}
	out := make([]string, int(r.nextID)+1)
	for i := 0; i <= int(r.nextID); i++ {
		out[i] = r.paths[FileID(i)]
	}
	return out
}

func (r *ImportResolver) internFile(path string) FileID {
	if id, ok := r.intern[path]; ok {
		return id
	}
	r.nextID++
	r.intern[path] = r.nextID
	return r.nextID
}

func (r *ImportResolver) resolve(importPath, parentPath string) (*importerResolverFrame, error) {
	parentFrame, err := r.createImporterResolverFrame(importPath, parentPath)
	if err != nil {
		return nil, err
	}
	for _, importNode := range parentFrame.Grammar.Imports {
		childFrame, err := r.resolve(importNode.GetPath(), parentFrame.ImportPath)
		if err != nil {
			return nil, err
		}
		for _, name := range importNode.GetNames() {
			importedDefinition, ok := childFrame.Grammar.DefsByName[name]
			if !ok {
				return nil, fmt.Errorf("Name `%s` isn't declared in %s", name, childFrame.ImportPath)
			}

			parentFrame.Grammar.AddDefinition(importedDefinition)

			deps := childFrame.findDefinitionDeps(importedDefinition)

			for _, depName := range deps.names {
				parentFrame.Grammar.AddDefinition(deps.nodes[depName])
			}
		}
	}

	parentFrame.Grammar.Imports = []*ImportNode{}

	return parentFrame, nil
}

func (r *ImportResolver) createImporterResolverFrame(importPath, parentPath string) (*importerResolverFrame, error) {
	path, err := r.loader.GetPath(importPath, parentPath)
	if err != nil {
		return nil, err
	}
	data, err := r.loader.GetContent(path)
	if err != nil {
		return nil, err
	}
	p := NewGrammarParser(data)

	id := r.internFile(path)
	r.paths[id] = path
	p.SetGrammarFile(path)
	p.SetGrammarFileID(id)

	node, err := p.Parse()
	if err != nil {
		return nil, err
	}
	grammar, ok := node.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", node)
	}
	f := &importerResolverFrame{
		ImportPath: path,
		Grammar:    grammar,
	}
	return f, nil
}

type importerResolverFrame struct {
	ImportPath string
	Grammar    *GrammarNode
}

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
func (f *importerResolverFrame) findDefinitionDeps(node *DefinitionNode) *sortedDeps {
	deps := newSortedDeps()
	findDefinitionDeps(f.Grammar, node.Expr, deps)
	return deps
}

func findDefinitionDeps(g *GrammarNode, node AstNode, deps *sortedDeps) {
	switch n := node.(type) {
	case *DefinitionNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *IdentifierNode:
		// Let's not recurse if this dep has been seen already
		if _, ok := deps.nodes[n.Value]; ok {
			return
		}

		// save definition as a dependency and recurse into it
		def := g.DefsByName[n.Value]
		deps.nodes[n.Value] = def
		deps.names = append(deps.names, n.Value)
		findDefinitionDeps(g, def.Expr, deps)
	case *SequenceNode:
		for _, item := range n.Items {
			findDefinitionDeps(g, item, deps)
		}
	case *ChoiceNode:
		findDefinitionDeps(g, n.Left, deps)
		findDefinitionDeps(g, n.Right, deps)
	case *OptionalNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *ZeroOrMoreNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *OneOrMoreNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *AndNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *NotNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *LexNode:
		findDefinitionDeps(g, n.Expr, deps)
	case *LabeledNode:
		// save definition as a dependency and recurse into it
		if def, ok := g.DefsByName[n.Label]; ok {
			deps.nodes[n.Label] = def
			deps.names = append(deps.names, n.Label)
			findDefinitionDeps(g, def.Expr, deps)
		}
		findDefinitionDeps(g, n.Expr, deps)
	}
}
