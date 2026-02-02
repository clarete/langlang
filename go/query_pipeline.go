package langlang

import "sort"

// TransformedGrammarQuery applies all grammar transformations
// (builtins, charsets, whitespace, captures) to a resolved grammar.
//
// This orchestrates the transformation pipeline:
//
//   - ResolvedImportsQuery
//   - WithCharsetsQuery
//   - WithWhitespaceQuery
//   - WithCapturesQuery
var TransformedGrammarQuery = &Query[FilePath, *GrammarNode]{
	Name:    "TransformedGrammar",
	Compute: computeTransformedGrammar,
}

func computeTransformedGrammar(db *Database, key FilePath) (*GrammarNode, error) {
	// The final transformation is WithCapturesQuery, which chains through
	// all previous transformations via their dependencies.
	return Get(db, WithCapturesQuery, key)
}

// WithCharsetsQuery optimizes character classes into efficient charsets.
// Depends on ResolvedImportsQuery.
var WithCharsetsQuery = &Query[FilePath, *GrammarNode]{
	Name:    "WithCharsets",
	Compute: computeWithCharsets,
}

func computeWithCharsets(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	if !db.Config().GetBool("grammar.add_charsets") {
		return grammar, nil
	}
	return AddCharsets(grammar)
}

// WithWhitespaceQuery injects automatic whitespace handling.
// Depends on WithCharsetsQuery.
var WithWhitespaceQuery = &Query[FilePath, *GrammarNode]{
	Name:    "WithWhitespace",
	Compute: computeWithWhitespace,
}

func computeWithWhitespace(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := Get(db, WithCharsetsQuery, key)
	if err != nil {
		return nil, err
	}
	if !db.Config().GetBool("grammar.handle_spaces") {
		return grammar, nil
	}
	result, err := InjectWhitespaces(grammar)
	if err != nil {
		return nil, err
	}
	return result.(*GrammarNode), nil
}

// WithCapturesQuery adds capture nodes for AST construction.
// Depends on WithWhitespaceQuery. This is the final transformation.
var WithCapturesQuery = &Query[FilePath, *GrammarNode]{
	Name:    "WithCaptures",
	Compute: computeWithCaptures,
}

func computeWithCaptures(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := Get(db, WithWhitespaceQuery, key)
	if err != nil {
		return nil, err
	}
	if !db.Config().GetBool("grammar.captures") {
		return grammar, nil
	}
	// AddCaptures returns (*GrammarNode, error) directly
	return AddCaptures(grammar, db.Config())
}

// RecursiveSetQuery computes the set of all recursive definitions in
// a grammar file.
var RecursiveSetQuery = &Query[FilePath, map[string]struct{}]{
	Name:    "RecursiveSet",
	Compute: computeRecursiveSet,
}

func computeRecursiveSet(db *Database, key FilePath) (map[string]struct{}, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	return getIsRecursiveFromGrammar(grammar), nil
}

type callGraph map[string]map[string]struct{}

func getIsRecursiveFromGrammar(g *GrammarNode) map[string]struct{} {
	cg := make(callGraph, len(g.Definitions))
	for _, d := range g.Definitions {
		if _, ok := cg[d.Name]; !ok {
			cg[d.Name] = make(map[string]struct{})
		}
		for _, id := range findIdentifiers(d.Expr) {
			cg[d.Name][id] = struct{}{}
		}
	}
	return cg.getIsRecursive()
}

func findIdentifiers(node AstNode) []string {
	var ids []string
	Inspect(node, func(n AstNode) bool {
		if id, ok := n.(*IdentifierNode); ok {
			ids = append(ids, id.Value)
		}
		return true
	})
	return ids
}

func (g callGraph) getIsRecursive() map[string]struct{} {
	var (
		stack   = []string{}
		onStack = map[string]bool{}
		visited = map[string]bool{}
		recurse = map[string]struct{}{}
		pos     = map[string]int{}
		dfs     func(string)
	)
	dfs = func(v string) {
		visited[v] = true
		pos[v] = len(stack)
		stack = append(stack, v)
		onStack[v] = true

		// Sort edges for deterministic traversal
		edges := make([]string, 0, len(g[v]))
		for w := range g[v] {
			edges = append(edges, w)
		}
		sort.Strings(edges)

		for _, w := range edges {
			if !visited[w] {
				dfs(w)
				continue
			}
			if onStack[w] {
				// back edge v -> w: mark the cycle path w..v
				for i := pos[w]; i < len(stack); i++ {
					recurse[stack[i]] = struct{}{}
				}
			}
		}
		stack = stack[:len(stack)-1]
		onStack[v] = false
	}

	// ensure all vertices (even isolated ones) are visited Sort
	// vertices for deterministic traversal order (Go map
	// iteration is randomized)
	vertices := make([]string, 0, len(g))
	for v := range g {
		vertices = append(vertices, v)
	}
	sort.Strings(vertices)

	for _, v := range vertices {
		if !visited[v] {
			dfs(v)
		}
	}
	return recurse
}

// IsRecursiveQuery determines if a definition is recursive (directly
// or indirectly calls itself).
var IsRecursiveQuery = &Query[DefKey, bool]{
	Name:    "IsRecursive",
	Compute: computeIsRecursive,
}

func computeIsRecursive(db *Database, key DefKey) (bool, error) {
	recursiveSet, err := Get(db, RecursiveSetQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}
	_, isRecursive := recursiveSet[key.Name]
	return isRecursive, nil
}

// Left Recursion Analysis Query

var IsLeftRecursiveQuery = &Query[DefKey, bool]{
	Name:    "IsLeftRecursive",
	Compute: computeIsLeftRecursive,
}

func computeIsLeftRecursive(db *Database, key DefKey) (bool, error) {
	grammar, err := Get(db, ResolvedImportsQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}
	def, ok := grammar.DefsByName[key.Name]
	if !ok {
		return false, nil
	}
	for _, alt := range flattenChoices(def.Expr) {
		if isLeftRecursiveAlternative(alt, key.Name) {
			return true, nil
		}
	}
	return false, nil
}

func flattenChoices(node AstNode) []AstNode {
	var (
		alternatives []AstNode
		flatten      func(n AstNode)
	)
	flatten = func(n AstNode) {
		switch x := n.(type) {
		case *ChoiceNode:
			flatten(x.Left)
			flatten(x.Right)
		case *CaptureNode:
			flatten(x.Expr)
		case *LexNode:
			flatten(x.Expr)
		default:
			alternatives = append(alternatives, n)
		}
	}
	flatten(node)
	return alternatives
}

// isLeftRecursiveAlternative checks if an alternative starts with a
// call to the given rule name (direct left recursion).
func isLeftRecursiveAlternative(alt AstNode, ruleName string) bool {
	first := getFirstExpr(alt)
	if id, ok := first.(*IdentifierNode); ok {
		return id.Value == ruleName
	}
	if prec, ok := first.(*PrecedenceNode); ok {
		if id, ok := prec.Expr.(*IdentifierNode); ok {
			return id.Value == ruleName
		}
	}
	return false
}

// getFirstExpr returns the first "terminal" or identifier in an
// expression, skipping through sequences and other structural nodes.
func getFirstExpr(node AstNode) AstNode {
	switch n := node.(type) {
	case *SequenceNode:
		if len(n.Items) > 0 {
			return getFirstExpr(n.Items[0])
		}
		return nil
	case *LexNode:
		return getFirstExpr(n.Expr)
	case *CaptureNode:
		return getFirstExpr(n.Expr)
	case *LabeledNode:
		return getFirstExpr(n.Expr)
	default:
		return node
	}
}

// DefSizeQuery computes the compiled size (in instructions) of a
// definition, used for inlining decisions.
var DefSizeQuery = &Query[DefKey, int]{
	Name:    "DefSize",
	Compute: computeDefSize,
}

func computeDefSize(db *Database, key DefKey) (int, error) {
	// Get the transformed grammar
	grammar, err := Get(db, TransformedGrammarQuery, FilePath(key.File))
	if err != nil {
		return 0, err
	}

	def, ok := grammar.DefsByName[key.Name]
	if !ok {
		return 0, nil
	}

	// Create a temporary compiler in dry-run mode to count
	// instructions
	file := db.FileIDToPath(def.SourceLocation().FileID)
	tmpc := newCompilerWithDB(db, file)
	tmpc.dryRun = true
	tmpc.grammarNode = grammar

	if err := def.Accept(tmpc); err != nil {
		return 0, err
	}

	return tmpc.cursor, nil
}

// Compilation Queries

// CompiledProgramQuery compiles a grammar file to a Program (IR).
var CompiledProgramQuery = &Query[FilePath, *Program]{
	Name:    "CompiledProgram",
	Compute: computeCompiledProgram,
}

func computeCompiledProgram(db *Database, key FilePath) (*Program, error) {
	// Get the transformed grammar
	grammar, err := Get(db, TransformedGrammarQuery, key)
	if err != nil {
		return nil, err
	}

	// Compile using the query-aware compiler
	return CompileWithDB(db, string(key), grammar)
}

// EncodedBytecodeQuery encodes a compiled program to bytecode.
var EncodedBytecodeQuery = &Query[FilePath, *Bytecode]{
	Name:    "EncodedBytecode",
	Compute: computeEncodedBytecode,
}

func computeEncodedBytecode(db *Database, key FilePath) (*Bytecode, error) {
	// Get the compiled program
	program, err := Get(db, CompiledProgramQuery, key)
	if err != nil {
		return nil, err
	}

	// Encode using the existing Encode function
	return Encode(program, db.Config()), nil
}
