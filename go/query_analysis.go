package langlang

import (
	"errors"
	"fmt"
)

// Import Errors Query

// ImportErrorKind discriminates the type of import error.
type ImportErrorKind int

const (
	ImportErrorMissingName  ImportErrorKind = iota // Name not found in source file
	ImportErrorFileNotFound                        // Import file doesn't exist
	ImportErrorParseFailure                        // Import file has syntax errors
)

// ImportErrorInfo holds information about an import error.
type ImportErrorInfo struct {
	Kind       ImportErrorKind
	Name       string         // The name that wasn't found (for MissingName)
	SourceFile string         // The file being imported from
	Message    string         // Error message (for FileNotFound/ParseFailure)
	Location   SourceLocation // Where the import statement is
}

// ImportErrorsQuery finds all import-related errors: missing names,
// file not found, and parse errors in imported files.
var ImportErrorsQuery = &Query[FilePath, []ImportErrorInfo]{
	Name:    "ImportErrors",
	Compute: computeImportErrors,
}

func computeImportErrors(db *Database, key FilePath) ([]ImportErrorInfo, error) {
	return computeImportErrorsRecursive(db, string(key), string(key), make(map[string]bool))
}

func computeImportErrorsRecursive(db *Database, importPath, parentPath string, visited map[string]bool) ([]ImportErrorInfo, error) {
	// Resolve the actual file path
	path, err := db.Loader().GetPath(importPath, parentPath)
	if err != nil {
		return nil, nil // Can't resolve entry file path - skip
	}

	// Avoid cycles
	if visited[path] {
		return nil, nil
	}
	visited[path] = true

	// Get the parsed grammar (before import resolution)
	grammar, err := Get(db, ParsedGrammarQuery, FilePath(path))
	if err != nil {
		return nil, nil // Parse error in current file - skip (caught by ParseErrorsQuery)
	}

	var importErrors []ImportErrorInfo

	// Check each import statement
	for _, importNode := range grammar.Imports {
		importedPath, err := db.Loader().GetPath(importNode.GetPath(), path)
		if err != nil {
			// File not found error
			importErrors = append(importErrors, ImportErrorInfo{
				Kind:       ImportErrorFileNotFound,
				SourceFile: importNode.GetPath(),
				Message:    err.Error(),
				Location:   importNode.SourceLocation(),
			})
			continue
		}

		// Get the parsed grammar of the imported file
		importedGrammar, err := Get(db, ParsedGrammarQuery, FilePath(importedPath))
		if err != nil {
			// Determine the kind of error and extract message
			var (
				kind ImportErrorKind
				msg  string
			)

			var loadErr *FileLoadError
			var grammarErr *GrammarError

			if errors.As(err, &loadErr) {
				// File could not be loaded (not found, permission denied, etc.)
				kind = ImportErrorFileNotFound
				msg = loadErr.Err.Error()
			} else if errors.As(err, &grammarErr) && len(grammarErr.Diagnostics) > 0 {
				// Parse error with diagnostics
				kind = ImportErrorParseFailure
				msg = grammarErr.Diagnostics[0].Message
			} else {
				// Other parse error
				kind = ImportErrorParseFailure
				msg = err.Error()
			}

			importErrors = append(importErrors, ImportErrorInfo{
				Kind:       kind,
				SourceFile: importNode.GetPath(),
				Message:    msg,
				Location:   importNode.SourceLocation(),
			})
			continue
		}

		// Check each imported name
		for _, name := range importNode.GetNames() {
			if _, ok := importedGrammar.DefsByName[name]; !ok {
				importErrors = append(importErrors, ImportErrorInfo{
					Kind:       ImportErrorMissingName,
					Name:       name,
					SourceFile: importNode.GetPath(),
					Location:   importNode.SourceLocation(),
				})
			}
		}

		// Recursively check imports in the imported file
		childErrors, err := computeImportErrorsRecursive(db, importNode.GetPath(), path, visited)
		if err != nil {
			continue
		}
		importErrors = append(importErrors, childErrors...)
	}

	return importErrors, nil
}

// CallGraphDataQuery builds a graph of which rules reference which.
// Used for: Find References, Call Hierarchy, unused rule detection
var CallGraphDataQuery = &Query[FilePath, *CallGraphData]{
	Name:    "CallGraphData",
	Compute: computeCallGraphData,
}

func computeCallGraphData(db *Database, key FilePath) (*CallGraphData, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}

	callers := make(map[string][]CallerInfo)
	callees := make(map[string][]string)

	for _, def := range grammar.Definitions {
		// Initialize callees for this definition
		if _, ok := callees[def.Name]; !ok {
			callees[def.Name] = []string{}
		}

		// Find all identifiers referenced in this definition
		Inspect(def.Expr, func(n AstNode) bool {
			if id, ok := n.(*IdentifierNode); ok {
				callees[def.Name] = append(callees[def.Name], id.Value)
				callers[id.Value] = append(callers[id.Value], CallerInfo{
					Name:     def.Name,
					Location: id.SourceLocation(),
				})
			}
			return true
		})
	}
	return &CallGraphData{
		Callers: callers,
		Callees: callees,
	}, nil
}

// UnusedRulesQuery finds rules that are never referenced.
// Used for: Diagnostics (warnings), code lens
var UnusedRulesQuery = &Query[FilePath, []string]{
	Name:    "UnusedRules",
	Compute: computeUnusedRules,
}

func computeUnusedRules(db *Database, key FilePath) ([]string, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}

	callGraph, err := Get(db, CallGraphDataQuery, key)
	if err != nil {
		return nil, err
	}

	var unused []string

	// Skip the first definition (entry point) and builtins
	builtins := map[string]bool{
		"Spacing": true, "Space": true, "EOF": true, "EOL": true,
	}

	for i, def := range grammar.Definitions {
		// Skip the entry point
		if i == 0 {
			continue
		}

		// Skip builtins
		if builtins[def.Name] {
			continue
		}

		// Check if this rule is called anywhere
		if callers, ok := callGraph.Callers[def.Name]; !ok || len(callers) == 0 {
			// Also check if it's used as a recovery rule
			recoveryRules, _ := Get(db, RecoveryRulesQuery, key)
			if info, ok := recoveryRules[def.Name]; ok && len(info.UsageLocs) > 0 {
				continue // Used as recovery rule
			}
			unused = append(unused, def.Name)
		}
	}

	return unused, nil
}

// UndefinedReferencesQuery finds identifiers with no definition.
// Used for: Diagnostics (errors)
var UndefinedReferencesQuery = &Query[FilePath, []IdentifierLocation]{
	Name:    "UndefinedReferences",
	Compute: computeUndefinedReferences,
}

func computeUndefinedReferences(db *Database, key FilePath) ([]IdentifierLocation, error) {
	idLocs, err := Get(db, IdentifierLocationsQuery, key)
	if err != nil {
		return nil, err
	}

	defLocs, err := Get(db, DefinitionLocationsQuery, key)
	if err != nil {
		return nil, err
	}

	var undefined []IdentifierLocation
	for _, idLoc := range idLocs {
		if idLoc.IsDefinition {
			continue
		}
		if _, ok := defLocs[idLoc.Name]; !ok {
			undefined = append(undefined, idLoc)
		}
	}

	return undefined, nil
}

// Diagnostics Query

// DiagnosticsQuery returns all errors/warnings for a file.
// Used for: publishDiagnostics, real-time error reporting
var DiagnosticsQuery = &Query[FilePath, []Diagnostic]{
	Name:    "Diagnostics",
	Compute: computeDiagnostics,
}

func computeDiagnostics(db *Database, key FilePath) ([]Diagnostic, error) {
	var (
		diagnostics []Diagnostic
		entryPath   = string(key)
	)
	parseErrors, err := Get(db, AllParseErrorsQuery, key)
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, parseErrors...)

	// try to compute the rest of the diagnostics even if there are parse
	// errors, as they may not be fatal

	sourceFiles, err := Get(db, SourceFilesQuery, key)
	if err != nil {
		return nil, err
	}

	// Helper to resolve file path from a SourceLocation
	resolvePath := func(loc SourceLocation) string {
		if int(loc.FileID) >= 0 && int(loc.FileID) < len(sourceFiles) {
			return sourceFiles[loc.FileID]
		}
		return entryPath
	}

	importErrors, err := Get(db, ImportErrorsQuery, key)
	if err != nil {
		return nil, err
	}
	for _, ie := range importErrors {
		var msg, code string
		switch ie.Kind {
		case ImportErrorMissingName:
			msg = fmt.Sprintf("Name '%s' is not declared in %s", ie.Name, ie.SourceFile)
			code = "missing-import"
		case ImportErrorFileNotFound:
			msg = fmt.Sprintf("Cannot find import '%s': %s", ie.SourceFile, ie.Message)
			code = "import-not-found"
		case ImportErrorParseFailure:
			msg = fmt.Sprintf("Failed to parse import '%s': %s", ie.SourceFile, ie.Message)
			code = "import-parse-error"
		}
		diagnostics = append(diagnostics, Diagnostic{
			Location: ie.Location,
			Severity: DiagnosticError,
			Message:  msg,
			Code:     code,
			FilePath: resolvePath(ie.Location),
		})
	}
	undefinedRefs, err := Get(db, UndefinedReferencesQuery, key)
	if err != nil {
		return nil, err
	}
	for _, ref := range undefinedRefs {
		diagnostics = append(diagnostics, Diagnostic{
			Location: ref.Location,
			Severity: DiagnosticError,
			Message:  fmt.Sprintf("Undefined rule '%s'", ref.Name),
			Code:     "undefined-rule",
			FilePath: resolvePath(ref.Location),
		})
	}
	unusedRules, err := Get(db, UnusedRulesQuery, key)
	if err != nil {
		return nil, err
	}
	defLocs, err := Get(db, DefinitionLocationsQuery, key)
	if err != nil {
		return nil, err
	}
	for _, ruleName := range unusedRules {
		if loc, ok := defLocs[ruleName]; ok {
			diagnostics = append(diagnostics, Diagnostic{
				Location: loc,
				Severity: DiagnosticWarning,
				Message:  fmt.Sprintf("Rule '%s' is defined but never used", ruleName),
				Code:     "unused-rule",
				FilePath: resolvePath(loc),
			})
		}
	}
	recoveryRules, err := Get(db, RecoveryRulesQuery, key)
	if err != nil {
		return nil, err
	}
	for _, info := range recoveryRules {
		if !info.HasRecovery {
			for _, usageLoc := range info.UsageLocs {
				diagnostics = append(diagnostics, Diagnostic{
					Location: usageLoc,
					Severity: DiagnosticWarning,
					Message:  fmt.Sprintf("Label '%s' has no recovery rule; parser will fail on this error", info.LabelName),
					Code:     "missing-recovery-rule",
					FilePath: resolvePath(usageLoc),
				})
			}
		}
	}
	loopRisks, err := Get(db, InfiniteLoopRisksQuery, key)
	if err != nil {
		return nil, err
	}
	for _, risk := range loopRisks {
		var msg string
		severity := DiagnosticWarning
		if risk.Definitive {
			severity = DiagnosticError
			if risk.ViaRule != "" {
				msg = fmt.Sprintf(
					"Infinite loop: body of '%s' always succeeds without consuming input because rule '%s' is nullable",
					risk.Operator, risk.ViaRule)
			} else {
				msg = fmt.Sprintf(
					"Infinite loop: body of '%s' always succeeds without consuming input",
					risk.Operator)
			}
		} else {
			if risk.ViaRule != "" {
				msg = fmt.Sprintf(
					"Possible infinite loop: body of '%s' can match empty because rule '%s' is nullable",
					risk.Operator, risk.ViaRule)
			} else {
				msg = fmt.Sprintf(
					"Possible infinite loop: body of '%s' can match empty",
					risk.Operator)
			}
		}
		diagnostics = append(diagnostics, Diagnostic{
			Location: risk.Location,
			Severity: severity,
			Message:  msg,
			Code:     "infinite-loop",
			FilePath: resolvePath(risk.Location),
		})
	}
	return diagnostics, nil
}

// Compiler Optimization Queries

// ErrorLabelsQuery collects all error labels (^Label) used in a
// grammar.  These are the labels that can trigger error recovery.
var ErrorLabelsQuery = &Query[FilePath, map[string]struct{}]{
	Name:    "ErrorLabels",
	Compute: computeErrorLabels,
}

func computeErrorLabels(db *Database, key FilePath) (map[string]struct{}, error) {
	grammar, err := Get(db, TransformedGrammarQuery, key)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]struct{})
	for _, def := range grammar.Definitions {
		Inspect(def.Expr, func(n AstNode) bool {
			if labeled, ok := n.(*LabeledNode); ok {
				labels[labeled.Label] = struct{}{}
			}
			return true
		})
	}
	return labels, nil
}

// DefinitionDepsQuery finds all transitive dependencies of a
// definition.  This returns the names of all rules that a definition
// directly or indirectly references.
var DefinitionDepsQuery = &Query[DefKey, []string]{
	Name:    "DefinitionDeps",
	Compute: computeDefinitionDeps,
}

func computeDefinitionDeps(db *Database, key DefKey) ([]string, error) {
	grammar, err := Get(db, ResolvedImportsQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}
	def, ok := grammar.DefsByName[key.Name]
	if !ok {
		return nil, nil
	}
	deps := newSortedDeps()
	if err := findDefinitionDeps(grammar, def.Expr, deps); err != nil {
		return nil, err
	}
	return deps.names, nil
}

// ShouldInlineQuery determines if a definition should be inlined.
//
// A definition is inlined if:
//
// - Inlining is enabled in config
// - It's not the entry point (first definition)
// - It's not used as a recovery rule (error label)
// - It's not recursive
// - Its compiled size is within the threshold
var ShouldInlineQuery = &Query[DefKey, bool]{
	Name:    "ShouldInline",
	Compute: computeShouldInline,
}

func computeShouldInline(db *Database, key DefKey) (bool, error) {
	cfg := db.Config()
	if !cfg.GetBool("compiler.inline.enabled") {
		return false, nil
	}
	grammar, err := Get(db, TransformedGrammarQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}
	if grammar.FirstDefinition() != nil && grammar.FirstDefinition().Name == key.Name {
		return false, nil
	}
	errorLabels, err := Get(db, ErrorLabelsQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}
	if _, isRecovery := errorLabels[key.Name]; isRecovery {
		return false, nil
	}
	isRecursive, err := Get(db, IsRecursiveQuery, key)
	if err != nil {
		return false, err
	}
	if isRecursive {
		return false, nil
	}
	size, err := Get(db, DefSizeQuery, key)
	if err != nil {
		return false, err
	}
	maxSize := cfg.GetInt("compiler.inline.max_size")
	return size <= maxSize, nil
}

// Expression Analysis Queries

// IsSyntacticQuery determines if a definition is syntactic (only
// matches terminals, no semantic actions or non-terminal references
// that could fail).
var IsSyntacticQuery = &Query[DefKey, bool]{
	Name:    "IsSyntactic",
	Compute: computeIsSyntactic,
}

func computeIsSyntactic(db *Database, key DefKey) (bool, error) {
	grammar, err := Get(db, TransformedGrammarQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}

	def, ok := grammar.DefsByName[key.Name]
	if !ok {
		return false, nil
	}

	return isSyntactic(def.Expr, true), nil
}

// CapExprSizeQuery computes the fixed capture size of an expression,
// if it has one.  Returns (size, true) if the expression always
// captures exactly 'size' bytes, or (0, false) if the size varies.
var CapExprSizeQuery = &Query[DefKey, *CapExprSizeResult]{
	Name:    "CapExprSize",
	Compute: computeCapExprSizeQuery,
}

type CapExprSizeResult struct {
	Size    int
	IsFixed bool
}

func computeCapExprSizeQuery(db *Database, key DefKey) (*CapExprSizeResult, error) {
	grammar, err := Get(db, TransformedGrammarQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}

	def, ok := grammar.DefsByName[key.Name]
	if !ok {
		return &CapExprSizeResult{Size: 0, IsFixed: false}, nil
	}

	size, isFixed := capExprSizeForNode(def.Expr)
	return &CapExprSizeResult{Size: size, IsFixed: isFixed}, nil
}

// capExprSizeForNode computes the fixed capture size of an expression.
func capExprSizeForNode(node AstNode) (int, bool) {
	switch n := node.(type) {
	case *CharsetNode:
		return 1, true

	case *LiteralNode:
		return len(n.Value), true

	case *ClassNode:
		val := -1
		for _, item := range n.Items {
			if is, ok := capExprSizeForNode(item); ok {
				if val >= 0 && val != is {
					return 0, false
				}
				val = is
			} else {
				return 0, false
			}
		}
		if val > 0 {
			return val, true
		}
		return 0, false

	case *ChoiceNode:
		left, ok := capExprSizeForNode(n.Left)
		if !ok {
			return 0, false
		}
		right, ok := capExprSizeForNode(n.Right)
		if !ok {
			return 0, false
		}
		if left == right {
			return left, true
		}
		return 0, false

	case *SequenceNode:
		var total int
		for _, item := range n.Items {
			if is, ok := capExprSizeForNode(item); ok {
				total += is
			} else {
				return 0, false
			}
		}
		if total > 0 {
			return total, true
		}
		return 0, false

	case *LexNode:
		return capExprSizeForNode(n.Expr)

	case *RangeNode:
		return 1, true

	default:
		return 0, false
	}
}

// String Table Query

// StringTableQuery computes the string table for a grammar.  The
// string table contains all strings used in the grammar (rule names,
// literals, error labels) with their interned IDs.
var StringTableQuery = &Query[FilePath, *StringTable]{
	Name:    "StringTable",
	Compute: computeStringTable,
}

type StringTable struct {
	Strings    []string
	StringsMap map[string]int
}

func computeStringTable(db *Database, key FilePath) (*StringTable, error) {
	grammar, err := Get(db, TransformedGrammarQuery, key)
	if err != nil {
		return nil, err
	}
	st := &StringTable{
		Strings:    []string{""}, // Reserve index 0 for "no name" sentinel
		StringsMap: make(map[string]int),
	}
	intern := func(s string) {
		if _, ok := st.StringsMap[s]; !ok {
			st.StringsMap[s] = len(st.Strings)
			st.Strings = append(st.Strings, s)
		}
	}
	for _, def := range grammar.Definitions {
		intern(def.Name)
	}
	for _, def := range grammar.Definitions {
		Inspect(def.Expr, func(n AstNode) bool {
			if labeled, ok := n.(*LabeledNode); ok {
				intern(labeled.Label)
			}
			return true
		})
	}
	return st, nil
}
