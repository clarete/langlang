package langlang

// QueryMatcher creates a Matcher from a grammar file using the query
// system for caching and incremental compilation.
func QueryMatcher(db *Database, entryPath string) (Matcher, error) {
	bytecode, err := Get(db, EncodedBytecodeQuery, FilePath(entryPath))
	if err != nil {
		return nil, err
	}

	vm := NewVirtualMachine(bytecode)
	vm.SetShowFails(db.Config().GetBool("vm.show_fails"))
	return vm, nil
}

// MatcherFromFilePath compiles a grammar from a file.
func MatcherFromFilePath(filePath string) (Matcher, error) {
	return MatcherFromFilePathWithConfig(filePath, NewConfig())
}

// MatcherFromFilePathWithConfig compiles a grammar from a file with a
// custom configuration.
func MatcherFromFilePathWithConfig(filePath string, cfg *Config) (Matcher, error) {
	loader := NewRelativeImportLoader()
	db := NewDatabase(cfg, loader)
	return QueryMatcher(db, filePath)
}

const matcherFromStringEntry = "grammar.peg"

// MatcherFromString compiles a single grammar source string into a
// Matcher using an in-memory loader. It is a convenience for one-off
// grammars without filesystem or multi-file imports.
func MatcherFromString(grammar string) (Matcher, error) {
	return MatcherFromBytes([]byte(grammar))
}

// MatcherFromBytes compiles a single grammar source (as bytes) into a
// Matcher using an in-memory loader. It is a convenience for one-off
// grammars without filesystem or multi-file imports.
func MatcherFromBytes(grammar []byte) (Matcher, error) {
	return MatcherFromBytesWithConfig(grammar, NewConfig())
}

func MatcherFromBytesWithConfig(grammar []byte, cfg *Config) (Matcher, error) {
	loader := NewInMemoryImportLoader()
	loader.Add(matcherFromStringEntry, grammar)
	db := NewDatabase(cfg, loader)
	return QueryMatcher(db, matcherFromStringEntry)
}

// QueryAST returns the transformed AST for a grammar file using the
// query system.
func QueryAST(db *Database, entryPath string) (*GrammarNode, error) {
	return Get(db, TransformedGrammarQuery, FilePath(entryPath))
}

// QueryProgram returns the compiled Program (IR) for a grammar file
// using the query system.
func QueryProgram(db *Database, entryPath string) (*Program, error) {
	return Get(db, CompiledProgramQuery, FilePath(entryPath))
}

// QueryBytecode returns the encoded bytecode for a grammar file using
// the query system.
func QueryBytecode(db *Database, entryPath string) (*Bytecode, error) {
	return Get(db, EncodedBytecodeQuery, FilePath(entryPath))
}

// QueryDiagnostics returns all diagnostics (errors, warnings, info)
// for a grammar file. This includes parse errors and semantic
// analysis results.
func QueryDiagnostics(db *Database, entryPath string) ([]Diagnostic, error) {
	return Get(db, DiagnosticsQuery, FilePath(entryPath))
}

// QueryDiagnosticsAsError returns diagnostics as a GrammarError if
// there are any error-level diagnostics.  Returns nil if there are no
// errors.  Warnings and info messages are included in the error but
// don't cause it to be non-nil on their own.
func QueryDiagnosticsAsError(db *Database, entryPath string) error {
	diagnostics, err := QueryDiagnostics(db, entryPath)
	if err != nil {
		return err
	}
	// Only return error if there are actual error-level diagnostics
	hasErrors := false
	for _, d := range diagnostics {
		if d.Severity == DiagnosticError {
			hasErrors = true
			break
		}
	}
	if !hasErrors {
		return nil
	}
	return NewGrammarError(diagnostics)
}

// QueryResolver provides a high-level API for resolving grammar
// imports and creating matchers, with query-based caching.
type QueryResolver struct {
	db *Database
}

// NewQueryResolver creates a new QueryResolver with the given
// database.
func NewQueryResolver(db *Database) *QueryResolver {
	return &QueryResolver{db: db}
}

// Resolve returns the transformed AST for a grammar file.
func (r *QueryResolver) Resolve(source string) (*GrammarNode, error) {
	return QueryAST(r.db, source)
}

// MatcherFor creates a Matcher for a grammar file.
func (r *QueryResolver) MatcherFor(entry string) (Matcher, error) {
	return QueryMatcher(r.db, entry)
}

// Invalidate invalidates all cached data for a file, forcing
// recomputation on the next query.
func (r *QueryResolver) Invalidate(path string) {
	r.db.InvalidateFile(path)
}

// InvalidateAll invalidates all cached data.
func (r *QueryResolver) InvalidateAll() {
	r.db.InvalidateAll()
}

// Stats returns statistics about the query cache.
func (r *QueryResolver) Stats() DatabaseStats {
	return r.db.Stats()
}

// Database returns the underlying query database.
func (r *QueryResolver) Database() *Database {
	return r.db
}
