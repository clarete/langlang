package langlang

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type ImportResolver struct {
	loader ImportLoader
}

func NewImportResolver(loader ImportLoader) *ImportResolver {
	return &ImportResolver{loader: loader}
}

func (r *ImportResolver) Resolve(source string) (AstNode, error) {
	f, err := r.resolve(source, source)
	if err != nil {
		return nil, err
	}
	return f.Grammar, nil
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
	p.SetGrammarFile(importPath)
	node, err := p.Parse()
	if err != nil {
		return nil, err
	}
	grammar, ok := node.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("Grammar expected, but got %#v", node)
	}
	f := &importerResolverFrame{
		ImportPath: path,
		Grammar:    grammar,
	}
	return f, nil
}

type ImportLoader interface {
	GetPath(importPath, parentPath string) (string, error)
	GetContent(path string) (string, error)
}

type RelativeImportLoader struct{}

func NewRelativeImportLoader() *RelativeImportLoader {
	return &RelativeImportLoader{}
}

func (ril *RelativeImportLoader) GetPath(importPath, parentPath string) (string, error) {
	// Root node handling
	if importPath == parentPath {
		return importPath, nil
	}
	var contents string
	if len(importPath) < 4 {
		return contents, fmt.Errorf("Path too short, it should start with ./: %s", importPath)
	}
	if importPath[:2] != "./" {
		return contents, fmt.Errorf("Path isn't relative to the import site: %s", importPath)
	}
	modulePath := importPath[2:]
	return filepath.Join(filepath.Dir(parentPath), modulePath), nil
}

func (ril *RelativeImportLoader) GetContent(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

type importerResolverFrame struct {
	ImportPath string
	Grammar    *GrammarNode
}

type sortedDeps struct {
	names []string
	nodes map[string]*DefinitionNode
}

// findDefinitionDeps traverses the definition `node` and finds all
// identifiers within it.  If the identifier hasn't been seen yet, it
// will add it to the dependency list, and traverse into the
// definition that points into that identifier.
func (f *importerResolverFrame) findDefinitionDeps(node *DefinitionNode) *sortedDeps {
	deps := &sortedDeps{names: []string{}, nodes: map[string]*DefinitionNode{}}
	f.doFindDefinitionDeps(node.Expr, deps)
	return deps
}

func (f *importerResolverFrame) doFindDefinitionDeps(node AstNode, deps *sortedDeps) {
	switch n := node.(type) {
	case *IdentifierNode:
		// Let's not recurse if this dep has been seen already
		if _, ok := deps.nodes[n.Value]; ok {
			return
		}

		// save definition as a dependency and recurse into it
		def := f.Grammar.DefsByName[n.Value]
		deps.nodes[n.Value] = def
		deps.names = append(deps.names, n.Value)
		f.doFindDefinitionDeps(def.Expr, deps)
	case *SequenceNode:
		for _, item := range n.Items {
			f.doFindDefinitionDeps(item, deps)
		}
	case *ChoiceNode:
		f.doFindDefinitionDeps(n.Left, deps)
		f.doFindDefinitionDeps(n.Right, deps)
	case *OptionalNode:
		f.doFindDefinitionDeps(n.Expr, deps)
	case *ZeroOrMoreNode:
		f.doFindDefinitionDeps(n.Expr, deps)
	case *OneOrMoreNode:
		f.doFindDefinitionDeps(n.Expr, deps)
	case *AndNode:
		f.doFindDefinitionDeps(n.Expr, deps)
	case *NotNode:
		f.doFindDefinitionDeps(n.Expr, deps)
	case *LexNode:
		f.doFindDefinitionDeps(n.Expr, deps)
	case *LabeledNode:
		// save definition as a dependency and recurse into it
		if def, ok := f.Grammar.DefsByName[n.Label]; ok {
			deps.nodes[n.Label] = def
			deps.names = append(deps.names, n.Label)
			f.doFindDefinitionDeps(def.Expr, deps)
		}
		f.doFindDefinitionDeps(n.Expr, deps)
	}
}
