package langlang

// ImportErrorKind discriminates the type of import error.
type ImportErrorKind int

const (
	ImportErrorNone ImportErrorKind = iota
	ImportErrorResolve
	ImportErrorCycle
	ImportErrorMissingName
)

type ImportGraph struct {
	// Root is the resolved path of the entry file
	Root string
	// Order contanis resolved paths in pre-order, with Root first
	Order []string
	// Files maps between resolved paths and parsed grammar nodes
	Files map[string]*GrammarNode
	// Edges maps between resolved paths and their import nodes.
	// The edges are sorted in the order they appear in the
	// source.
	Edges map[string][]ImportEdge
}

type ImportEdge struct {
	Node     *ImportNode
	From, To string
	ErrMsg   string
	ErrKind  ImportErrorKind
}

func (e ImportEdge) IsCycle() bool {
	return e.ErrKind == ImportErrorCycle
}

var ImportGraphQuery = &Query[FilePath, *ImportGraph]{
	Name:    "ImportGraph",
	Compute: computeImportGraph,
}

func computeImportGraph(db *Database, key FilePath) (*ImportGraph, error) {
	root, err := db.Loader().GetPath(string(key), string(key))
	if err != nil {
		return nil, err
	}
	g := &ImportGraph{
		Root:  root,
		Files: map[string]*GrammarNode{},
		Edges: map[string][]ImportEdge{},
	}
	if err := g.visit(db, root, map[string]bool{}); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *ImportGraph) visit(db *Database, path string, inProgress map[string]bool) error {
	if _, seen := g.Files[path]; seen {
		return nil
	}
	grammar, err := Get(db, ParsedGrammarQuery, FilePath(path))
	if err != nil {
		return err
	}
	g.Files[path] = grammar
	g.Order = append(g.Order, path)

	inProgress[path] = true
	defer delete(inProgress, path)

	for _, node := range grammar.Imports {
		edge := ImportEdge{Node: node, From: path}
		to, err := db.Loader().GetPath(node.GetPath(), path)
		switch {
		case err != nil:
			edge.ErrKind = ImportErrorResolve
			edge.ErrMsg = err.Error()
		case inProgress[to]:
			edge.To = to
			edge.ErrKind = ImportErrorCycle
		default:
			if _, err := Get(db, ParsedGrammarQuery, FilePath(to)); err != nil {
				edge.ErrKind = ImportErrorResolve
				edge.ErrMsg = err.Error()
			} else {
				edge.To = to
			}
		}
		g.Edges[path] = append(g.Edges[path], edge)
		if edge.To != "" && !edge.IsCycle() {
			if err := g.visit(db, to, inProgress); err != nil {
				return err
			}
		}
	}
	return nil
}
