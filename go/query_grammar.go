package langlang

import "fmt"

// ParsedGrammarQuery parses a grammar file and returns its AST.
// The returned grammar may contain ErrorNode instances if there were
// parse errors - check grammar.Errors to get diagnostics.
var ParsedGrammarQuery = &Query[FilePath, *GrammarNode]{
	Name:    "ParsedGrammar",
	Compute: computeParsedGrammar,
}

// computeParsedGrammar parses a grammar file and returns its AST.
// This is a leaf query that reads from the filesystem.  The FileID is
// assigned from the database's file registry, ensuring stable IDs
// across all queries.
//
// Even if parsing fails, this returns a partial grammar with errors
// collected in the Errors field. This allows downstream queries to
// work with partial ASTs and report all errors at once.
func computeParsedGrammar(db *Database, key FilePath) (*GrammarNode, error) {
	path := string(key)

	data, err := db.Loader().GetContent(path)
	if err != nil {
		return nil, &FileLoadError{Path: path, Err: err}
	}

	fileID := db.InternFileID(path)
	p := NewGrammarParser(data)
	p.SetGrammarFile(path)
	p.SetGrammarFileID(fileID)
	p.SetShowFails(true) // grammar parser always shows attempted matches

	// Parse always returns a grammar now, possibly with errors.
	// the following two checks are out of caution in case
	// something changes in the future, or while debugging
	// bootstrapping changes for the langlang.peg grammar file.
	node, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	grammar, ok := node.(*GrammarNode)
	if !ok {
		return nil, fmt.Errorf("grammar expected, but got %#v", node)
	}
	return grammar, nil
}

// Import Resolution Queries

// ResolvedImportsQuery resolves all imports for a grammar file,
// returning a complete grammar with all imported definitions merged.
var ResolvedImportsQuery = &Query[FilePath, *GrammarNode]{
	Name:    "ResolvedImports",
	Compute: computeResolvedImports,
}

// computeResolvedImports resolves all imports for a grammar file,
// returning a complete grammar with all imported definitions merged.
//
// If builtins are enabled, they are merged as an implicit import from
// BuiltinsPath, giving them proper FileIDs for code navigation.
func computeResolvedImports(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := resolveImportsRecursive(db, string(key), string(key))
	if err != nil {
		return nil, err
	}
	if db.Config().GetBool("grammar.add_builtins") {
		builtinsGrammar, err := resolveImportsRecursive(db, BuiltinsPath, BuiltinsPath)
		if err != nil {
			return nil, err
		}
		grammar = mergeBuiltinsGrammar(grammar, builtinsGrammar)
	}
	grammar.SourceFiles = db.AllFilePaths()
	return grammar, nil
}

// mergeBuiltinsGrammar merges builtin definitions into the user
// grammar.  User definitions take precedence: if a user defines a
// rule with the same name as a builtin, the builtin is not added.
func mergeBuiltinsGrammar(grammar, builtins *GrammarNode) *GrammarNode {
	grammar = copyGrammarNode(grammar)
	for _, def := range builtins.Definitions {
		if _, exists := grammar.DefsByName[def.Name]; !exists {
			grammar.AddDefinition(def)
		}
	}
	return grammar
}

// resolveImportsRecursive recursively resolves imports for a grammar
// file.  FileIDs are now set during parsing via computeParsedGrammar,
// so we don't need to update them here.
func resolveImportsRecursive(db *Database, importPath, parentPath string) (*GrammarNode, error) {
	path, err := db.Loader().GetPath(importPath, parentPath)
	if err != nil {
		return nil, err
	}
	grammar, err := Get(db, ParsedGrammarQuery, FilePath(path))
	if err != nil {
		return nil, err
	}

	grammar = copyGrammarNode(grammar)

	for _, importNode := range grammar.Imports {
		childGrammar, err := resolveImportsRecursive(db, importNode.GetPath(), path)
		if err != nil {
			// These errors will be surfaced by the ParsedGrammarQuery query
			continue
		}

		for _, name := range importNode.GetNames() {
			importedDefinition, ok := childGrammar.DefsByName[name]
			if !ok {
				// Skip missing names: they'll be caught as undefined
				// references by UndefinedReferencesQuery when used
				continue
			}

			grammar.AddDefinition(importedDefinition)

			deps, err := findDefinitionDepsFromGrammar(childGrammar, importedDefinition)
			if err != nil {
				// Skip deps that fail as they will be caught by
				// UndefinedReferencesQuery later
				continue
			}
			for _, depName := range deps.names {
				grammar.AddDefinition(deps.nodes[depName])
			}
		}
	}

	grammar.Imports = []*ImportNode{}

	return grammar, nil
}

// copyGrammarNode creates a shallow copy of a grammar node to avoid
// mutating cached values.
func copyGrammarNode(g *GrammarNode) *GrammarNode {
	defs := make([]*DefinitionNode, len(g.Definitions))
	copy(defs, g.Definitions)

	defsByName := make(map[string]*DefinitionNode, len(g.DefsByName))
	for k, v := range g.DefsByName {
		defsByName[k] = v
	}

	imports := make([]*ImportNode, len(g.Imports))
	copy(imports, g.Imports)

	return &GrammarNode{
		src:         g.src,
		Imports:     imports,
		Definitions: defs,
		DefsByName:  defsByName,
		SourceFiles: g.SourceFiles,
	}
}

func findDefinitionDepsFromGrammar(g *GrammarNode, node *DefinitionNode) (*sortedDeps, error) {
	deps := newSortedDeps()
	if err := findDefinitionDeps(g, node.Expr, deps); err != nil {
		return nil, err
	}
	return deps, nil
}

// SourceFilesQuery returns the list of source files involved in
// compiling a grammar (including imports).
var SourceFilesQuery = &Query[FilePath, []string]{
	Name:    "SourceFiles",
	Compute: computeSourceFiles,
}

// computeSourceFiles returns the list of source files involved in
// compiling a grammar.  It delegates to the database's file registry.
func computeSourceFiles(db *Database, key FilePath) ([]string, error) {
	// Ensure imports are resolved first (this populates the file registry)
	_, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	return db.AllFilePaths(), nil
}
