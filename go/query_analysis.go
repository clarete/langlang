package langlang

import "fmt"

// Import Errors Query

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
	ig, err := Get(db, ImportGraphQuery, key)
	if err != nil {
		return nil, err
	}

	var importErrors []ImportErrorInfo

	for _, path := range ig.Order {
		for _, edge := range ig.Edges[path] {
			if edge.ErrKind != ImportErrorNone {
				importErrors = append(importErrors, ImportErrorInfo{
					Kind:       edge.ErrKind,
					Message:    edge.ErrMsg,
					SourceFile: edge.Node.GetPath(),
					Location:   edge.Node.SourceLocation(),
				})
				continue
			}

			// Check each imported name
			for _, name := range edge.Node.GetNames() {
				if _, ok := ig.Files[edge.To].DefsByName[name]; !ok {
					importErrors = append(importErrors, ImportErrorInfo{
						Kind:       ImportErrorMissingName,
						Name:       name,
						SourceFile: edge.Node.GetPath(),
						Location:   edge.Node.SourceLocation(),
					})
				}
			}
		}
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
	rg, err := Get(db, RuleGraphQuery, key)
	if err != nil {
		return nil, err
	}

	grammar := rg.Grammar()
	callers := make(map[string][]CallerInfo)
	callees := make(map[string][]string)

	for _, def := range grammar.Definitions {
		callees[def.Name] = []string{}
		for _, ref := range rg.refs[def.Name] {
			if ref.Kind != RefKind_Call {
				continue
			}
			callees[def.Name] = append(callees[def.Name], ref.Name)
			callers[ref.Name] = append(callers[ref.Name], CallerInfo{
				Name:     def.Name,
				Location: ref.Loc,
			})
		}
	}
	return &CallGraphData{
		Callers: callers,
		Callees: callees,
	}, nil
}

var RuleGraphQuery = &Query[FilePath, *RuleGraph]{
	Name:    "RuleGraph",
	Compute: computeRuleGraph,
}

func computeRuleGraph(db *Database, key FilePath) (*RuleGraph, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	return newRuleGraph(grammar), nil
}

// UnusedRulesQuery finds rules that are never referenced.
// Used for: Diagnostics (warnings), code lens
var UnusedRulesQuery = &Query[FilePath, []string]{
	Name:    "UnusedRules",
	Compute: computeUnusedRules,
}

func computeUnusedRules(db *Database, key FilePath) ([]string, error) {
	rg, err := Get(db, RuleGraphQuery, key)
	if err != nil {
		return nil, err
	}
	var (
		grammar  = rg.Grammar()
		callers  = rg.Callers()
		implicit = rg.SpacingClosure()
		unused   []string
	)
	for i, def := range grammar.Definitions {
		if i == 0 {
			continue // Skip the entry point
		}
		if isBuiltinDefinition(grammar, def) {
			continue
		}
		if _, ok := implicit[def.Name]; ok {
			continue
		}
		if len(callers[def.Name]) == 0 {
			unused = append(unused, def.Name)
		}
	}
	return unused, nil
}

// isBuiltinDefinition tries to find out if a definition is builtin or
// user-defined by matching the file name it was declared.
func isBuiltinDefinition(g *GrammarNode, def *DefinitionNode) bool {
	id := int(def.SourceLocation().FileID)
	if id < len(g.SourceFiles) {
		return g.SourceFiles[id] == BuiltinsPath
	}
	return false
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
		case ImportErrorResolve:
			msg = fmt.Sprintf("Cannot find import '%s': %s", ie.SourceFile, ie.Message)
			code = "import-not-found"
		case ImportErrorCycle:
			msg = fmt.Sprintf("Import cycle detected: '%s'", ie.SourceFile)
			if ie.Message != "" {
				msg += ": " + ie.Message
			}
			code = "import-cycle"
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
