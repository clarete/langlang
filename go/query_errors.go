package langlang

import (
	"fmt"
	"strings"
)

// ParseErrorsQuery returns parse errors for a single grammar file.
// This includes both fatal parse errors (if the parser fails
// completely) and recovered errors (if the parser uses error
// recovery).  Note: Use AllParseErrorsQuery to get errors from all
// imported files.
var ParseErrorsQuery = &Query[FilePath, []Diagnostic]{
	Name:    "ParseErrors",
	Compute: computeParseErrors,
}

// AllParseErrorsQuery returns parse errors from the entry file and
// all imported files. This provides complete error coverage for
// grammars with imports.
var AllParseErrorsQuery = &Query[FilePath, []Diagnostic]{
	Name:    "AllParseErrors",
	Compute: computeAllParseErrors,
}

func computeParseErrors(db *Database, key FilePath) ([]Diagnostic, error) {
	path := string(key)

	// Get the parsed grammar - it now contains ErrorNode instances
	grammar, err := Get(db, ParsedGrammarQuery, key)
	if err != nil {
		// This shouldn't happen anymore since ParsedGrammarQuery
		// now always returns a grammar (possibly with errors)
		return nil, err
	}

	// Convert ErrorNode instances to Diagnostics
	var diagnostics []Diagnostic
	for _, errNode := range grammar.Errors {
		diagnostics = append(diagnostics, errorNodeToDiagnostic(errNode, path))
	}

	return diagnostics, nil
}

// computeAllParseErrors collects parse errors from the entry file and
// all files it imports (transitively).
func computeAllParseErrors(db *Database, key FilePath) ([]Diagnostic, error) {
	var allDiagnostics []Diagnostic

	entryErrors, err := Get(db, ParseErrorsQuery, key)
	if err != nil {
		return nil, err
	}
	allDiagnostics = append(allDiagnostics, entryErrors...)

	if len(entryErrors) > 0 {
		for _, d := range entryErrors {
			// If entry file has fatal parse errors, we
			// can't discover imports
			if d.Code == "syntax-error" || d.Code == "parse-error" {
				return allDiagnostics, nil
			}
		}
	}
	importedFiles, err := discoverImportedFiles(db, string(key), string(key), make(map[string]bool))
	if err != nil {
		// If import discovery fails, return what we have
		return allDiagnostics, nil
	}
	for _, importPath := range importedFiles {
		fileErrors, err := Get(db, ParseErrorsQuery, FilePath(importPath))
		if err != nil {
			// Add an error diagnostic for the import failure
			allDiagnostics = append(allDiagnostics, Diagnostic{
				Location: SourceLocation{
					FileID: -1,
					Span:   Span{Start: NewLocation(1, 1, 0), End: NewLocation(1, 1, 0)},
				},
				Severity: DiagnosticError,
				Message:  fmt.Sprintf("Failed to check imported file '%s': %v", importPath, err),
				Code:     "import-error",
				FilePath: string(key),
			})
			continue
		}
		allDiagnostics = append(allDiagnostics, fileErrors...)
	}

	return allDiagnostics, nil
}

// discoverImportedFiles recursively discovers all imported files.
func discoverImportedFiles(db *Database, importPath, parentPath string, visited map[string]bool) ([]string, error) {
	path, err := db.Loader().GetPath(importPath, parentPath)
	if err != nil {
		return nil, err
	}
	if visited[path] {
		return nil, nil
	}
	visited[path] = true

	grammar, err := Get(db, ParsedGrammarQuery, FilePath(path))
	if err != nil {
		return nil, err
	}
	var discovered []string
	if path != parentPath || importPath != parentPath {
		discovered = append(discovered, path)
	}
	for _, imp := range grammar.Imports {
		childPaths, err := discoverImportedFiles(db, imp.GetPath(), path, visited)
		if err != nil {
			return nil, err
		}
		discovered = append(discovered, childPaths...)
	}
	return discovered, nil
}

// errorNodeToDiagnostic converts an ErrorNode AST node to a Diagnostic.
func errorNodeToDiagnostic(errNode *ErrorNode, filePath string) Diagnostic {
	message := errNode.Message
	if message == "" {
		message = fmt.Sprintf("Syntax error: %s", errNode.Code)
	}

	return Diagnostic{
		Location: errNode.SourceLocation(),
		Severity: DiagnosticError,
		Message:  message,
		Code:     errNode.Code,
		FilePath: filePath,
		Expected: errNode.Expected,
	}
}

// labelToErrorCode converts error labels to diagnostic error codes.
func labelToErrorCode(label string) string {
	switch label {
	case "MissingClosingParen", "MissingClosingCurly", "MissingClosingBracket",
		"MissingClosingSQuote", "MissingClosingDQuote":
		return "unclosed-delimiter"
	case "MissingLabel":
		return "missing-label"
	case "MissingImportName", "MissingImportFrom", "MissingImportSrc":
		return "invalid-import"
	case "MissingRightRange":
		return "invalid-range"
	default:
		return "syntax-error"
	}
}

// GrammarError represents one or more errors found during grammar
// parsing or analysis.  It provides rich location information
// including file paths and line/column numbers.
type GrammarError struct {
	Diagnostics []Diagnostic
}

// Error implements the error interface, formatting all diagnostics.
func (e *GrammarError) Error() string {
	if len(e.Diagnostics) == 0 {
		return "grammar error (no details)"
	}
	if len(e.Diagnostics) == 1 {
		return e.Diagnostics[0].FormatCLI()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d errors found:\n", len(e.Diagnostics))
	for _, d := range e.Diagnostics {
		b.WriteString("  ")
		b.WriteString(d.FormatCLI())
		b.WriteRune('\n')
	}
	return b.String()
}

// HasErrors returns true if there are any error-level diagnostics.
func (e *GrammarError) HasErrors() bool {
	for _, d := range e.Diagnostics {
		if d.Severity == DiagnosticError {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of error-level diagnostics.
func (e *GrammarError) ErrorCount() int {
	count := 0
	for _, d := range e.Diagnostics {
		if d.Severity == DiagnosticError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-level diagnostics.
func (e *GrammarError) WarningCount() int {
	count := 0
	for _, d := range e.Diagnostics {
		if d.Severity == DiagnosticWarning {
			count++
		}
	}
	return count
}

// NewGrammarError creates a GrammarError from a list of diagnostics.
// Returns nil if there are no diagnostics.
func NewGrammarError(diagnostics []Diagnostic) error {
	if len(diagnostics) == 0 {
		return nil
	}
	return &GrammarError{Diagnostics: diagnostics}
}
