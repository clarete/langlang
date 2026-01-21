package langlang

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

// Note: getIsRecursiveFromGrammar is defined in grammar_compiler.go
// and is reused here. callGraph type and getIsRecursive method are
// also defined there.

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

	// Create a temporary compiler in dry-run mode to count instructions
	tmpc := newCompiler(db.Config())
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
