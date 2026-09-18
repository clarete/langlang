package langlang

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseErrorsQuery_ValidGrammar(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("valid.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, ParseErrorsQuery, FilePath("valid.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("expected no parse errors, got %d", len(diagnostics))
		for _, d := range diagnostics {
			t.Logf("  - %s", d.FormatCLI())
		}
	}
}

func TestImportErrorsQuery_FileNotFound(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("a.peg", []byte(`
@import B from "./nope.peg"
A <- B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, ImportErrorResolve, errs[0].Kind)
	assert.Equal(t, "./nope.peg", errs[0].SourceFile)
	assert.NotEmpty(t, errs[0].Message)
}

func TestImportErrorsQuery_MissingName(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("lib.peg", []byte(`B <- "b"`))
	loader.Add("a.peg", []byte(`
@import Nope from "./lib.peg"
A <- Nope
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, ImportErrorMissingName, errs[0].Kind)
	assert.Equal(t, "Nope", errs[0].Name)
}

func TestImportErrorsQuery_NestedImportErrors(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("c.peg", []byte(`C <- "c"`))
	loader.Add("b.peg", []byte(`
@import Missing from "./c.peg"
@import X from "./ghost.peg"
B <- "b"
`))
	loader.Add("a.peg", []byte(`
@import B from "./b.peg"
A <- B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// Errors inside an imported file's own imports must surface
	// when querying the root.
	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)

	kinds := map[ImportErrorKind]int{}
	for _, e := range errs {
		kinds[e.Kind]++
	}
	assert.Equal(t, 1, kinds[ImportErrorMissingName], "missing name in nested import: %v", errs)
	assert.Equal(t, 1, kinds[ImportErrorResolve], "unresolvable file in nested import: %v", errs)
}

func TestImportErrorsQuery_CleanImports(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("c.peg", []byte(`C <- "c"`))
	loader.Add("b.peg", []byte(`
@import C from "./c.peg"
B <- C
`))
	loader.Add("a.peg", []byte(`
@import B from "./b.peg"
A <- B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)
	assert.Empty(t, errs)
}

func TestImportErrorsQuery_CycleReported(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("a.peg", []byte(`
@import B from "./b.peg"
A <- B
`))
	loader.Add("b.peg", []byte(`
@import A from "./a.peg"
B <- "b"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// A cyclic import must terminate and be reported as a cycle -
	// exactly once, on the back-edge that closes the loop (b.peg's
	// import of a.peg). The forward edge (a -> b) is healthy and
	// must not produce a spurious error.
	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)

	require.Len(t, errs, 1)
	assert.Equal(t, ImportErrorCycle, errs[0].Kind)
	assert.Equal(t, "./a.peg", errs[0].SourceFile)
}

func TestImportErrorsQuery_ReExportedName(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// c.peg defines X; b.peg re-exports X from c; a.peg imports X
	// from b.
	loader.Add("c.peg", []byte(`X <- "x"`))
	loader.Add("b.peg", []byte(`
@import X from "./c.peg"
B <- X
`))
	loader.Add("a.peg", []byte(`
@import X from "./b.peg"
A <- X
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// X is not defined directly in b.peg, it is re-exported.  So
	// the name-check must consult b's *resolved* grammar, the same
	// view resolution uses.  Checking b's raw parsed grammar would
	// wrongly report X as a missing import.
	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)
	assert.Empty(t, errs)

	// The query must agree with resolution, which does provide X.
	g, err := Get(db, ResolvedImportsQuery, FilePath("a.peg"))
	require.NoError(t, err)
	assert.Contains(t, g.DefsByName, "X")
}

func TestImportErrorsQuery_ImportedBuiltinIsNotAnExport(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// lib.peg provides Y, but not EOF.  EOF is a builtin, injected
	// into every resolved grammar - but a file does not *export* a
	// builtin, so importing EOF from lib must still be missing.
	loader.Add("lib.peg", []byte(`Y <- "y"`))
	loader.Add("a.peg", []byte(`
@import EOF from "./lib.peg"
A <- EOF
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", true) // builtins in play
	db := NewDatabase(cfg, loader)

	// The name-check must use the builtins-free resolved view
	// (resolveFromGraph), not the fully-resolved-with-builtins
	// grammar, otherwise EOF would be found as an ambient builtin
	// and the real error masked.
	errs, err := Get(db, ImportErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, ImportErrorMissingName, errs[0].Kind)
	assert.Equal(t, "EOF", errs[0].Name)
}

// diagnosticByCode returns the single diagnostic carrying the given code,
// failing the test if there is not exactly one.  Import diagnostics
// are the user-facing surface, so tests assert on them rather than on
// the intermediate ImportErrorInfo.
func diagnosticByCode(t *testing.T, ds []Diagnostic, code string) Diagnostic {
	t.Helper()
	var found []Diagnostic
	for _, d := range ds {
		if d.Code == code {
			found = append(found, d)
		}
	}
	require.Lenf(t, found, 1, "expected exactly one %q diagnostic, got: %v", code, ds)
	return found[0]
}

func TestDiagnosticsQuery_ImportFileNotFound(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("a.peg", []byte(`
@import B from "./nope.peg"
A <- B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// A missing file shows up as a diagnostic on the import
	// statement, attributed to the importing file, carrying the
	// underlying loader error in its message.
	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("a.peg"))
	require.NoError(t, err)

	d := diagnosticByCode(t, diagnostics, "import-not-found")
	assert.Equal(t, DiagnosticError, d.Severity)
	assert.Equal(t, "a.peg", d.FilePath)
	assert.Contains(t, d.Message, "Cannot find import './nope.peg'")
	assert.Contains(t, d.Message, "nope.peg")
	// The diagnostic spans the import statement, not a zero range.
	assert.Greater(t, d.Location.Span.End.Cursor, d.Location.Span.Start.Cursor)
}

func TestDiagnosticsQuery_MissingImportName(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("lib.peg", []byte(`B <- "b"`))
	loader.Add("a.peg", []byte(`
@import Nope from "./lib.peg"
A <- Nope
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("a.peg"))
	require.NoError(t, err)

	// The message names both the symbol and the file it was
	// expected in - the source, not the importer.
	d := diagnosticByCode(t, diagnostics, "missing-import")
	assert.Equal(t, DiagnosticError, d.Severity)
	assert.Equal(t, "a.peg", d.FilePath)
	assert.Equal(t, "Name 'Nope' is not declared in ./lib.peg", d.Message)
	assert.Greater(t, d.Location.Span.End.Cursor, d.Location.Span.Start.Cursor)
}

func TestDiagnosticsQuery_ImportCycle(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("a.peg", []byte(`
@import B from "./b.peg"
A <- B
`))
	loader.Add("b.peg", []byte(`
@import A from "./a.peg"
B <- "b"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("a.peg"))
	require.NoError(t, err)

	// A cycle is reported once, on the back-edge import that closes
	// the loop - which lives in b.peg, so the diagnostic is
	// attributed there, not to the entry file.
	d := diagnosticByCode(t, diagnostics, "import-cycle")
	assert.Equal(t, DiagnosticError, d.Severity)
	assert.Equal(t, "b.peg", d.FilePath)
	assert.Contains(t, d.Message, "Import cycle detected")
	assert.Contains(t, d.Message, "./a.peg")
	assert.Greater(t, d.Location.Span.End.Cursor, d.Location.Span.Start.Cursor)
}

func TestAllParseErrorsQuery_BrokenImportDoesNotTruncateOthers(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// One import can't be found at all, the other has a syntax
	// error inside.  And the unresolvable one won't stop parse
	// errors from the other from being reported.
	loader.Add("a.peg", []byte(`
@import X from "./ghost.peg"
@import Helper from "./helper.peg"
A <- Helper
`))
	loader.Add("helper.peg", []byte(`Helper <- "hello`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, AllParseErrorsQuery, FilePath("a.peg"))
	require.NoError(t, err)

	found := false
	for _, d := range diagnostics {
		if d.FilePath == "helper.peg" {
			found = true
		}
	}
	assert.True(t, found, "expected helper.peg parse errors despite broken sibling import: %v", diagnostics)
}

func TestParseErrorsQuery_SyntaxError(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Missing closing quote
	loader.Add("invalid.peg", []byte(`G <- "hello`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, ParseErrorsQuery, FilePath("invalid.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diagnostics) == 0 {
		t.Fatal("expected parse errors, got none")
	}

	// Check that the error has location info
	d := diagnostics[0]
	if d.Severity != DiagnosticError {
		t.Errorf("expected error severity, got %v", d.Severity)
	}
	if d.FilePath != "invalid.peg" {
		t.Errorf("expected file path 'invalid.peg', got '%s'", d.FilePath)
	}
	if d.Location.Span.Start.Line < 1 {
		t.Errorf("expected valid line number, got %d", d.Location.Span.Start.Line)
	}
}

func TestParseErrorsQuery_RecoveredError(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Missing closing parenthesis - the grammar parser has error recovery for this
	loader.Add("recovered.peg", []byte(`G <- ("hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, ParseErrorsQuery, FilePath("recovered.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least one error (the missing paren)
	if len(diagnostics) == 0 {
		t.Fatal("expected recovered parse errors, got none")
	}

	// The error should indicate a missing closing paren
	found := false
	for _, d := range diagnostics {
		if strings.Contains(d.Code, "unclosed") || strings.Contains(d.Message, "paren") {
			found = true
			break
		}
	}
	if !found {
		t.Logf("diagnostics: %v", diagnostics)
		t.Error("expected error about unclosed delimiter")
	}
}

func TestDiagnosticsQuery_IncludesParseErrors(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Invalid grammar
	loader.Add("invalid.peg", []byte(`G <- "hello`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("invalid.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics, got none")
	}

	// Should have a syntax error
	hasParseError := false
	for _, d := range diagnostics {
		if d.Code == "syntax-error" || d.Code == "unclosed-delimiter" {
			hasParseError = true
			break
		}
	}
	if !hasParseError {
		t.Error("expected parse error in diagnostics")
	}
}

func TestDiagnosticsQuery_SemanticAfterParse(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Valid syntax but undefined reference
	loader.Add("semantic.peg", []byte(`G <- UndefinedRule`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("semantic.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have an undefined rule error
	found := false
	for _, d := range diagnostics {
		if d.Code == "undefined-rule" {
			found = true
			if d.FilePath != "semantic.peg" {
				t.Errorf("expected file path 'semantic.peg', got '%s'", d.FilePath)
			}
			break
		}
	}
	if !found {
		t.Error("expected undefined-rule error in diagnostics")
	}
}

func TestDiagnosticsQuery_AllDiagnosticsHaveFilePath(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Grammar with multiple potential issues
	loader.Add("multi.peg", []byte(`
G <- A B
A <- "a"
UnusedRule <- "unused"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("multi.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, d := range diagnostics {
		if d.FilePath == "" {
			t.Errorf("diagnostic has empty FilePath: %s", d.FormatCLI())
		}
		if d.Location.Span.Start.Line < 1 {
			t.Errorf("diagnostic has invalid line number: %s", d.FormatCLI())
		}
	}
}

func TestGrammarError_Formatting(t *testing.T) {
	diagnostics := []Diagnostic{
		{
			Location: SourceLocation{
				Span: Span{
					Start: NewLocation(10, 5, 100),
					End:   NewLocation(10, 15, 110),
				},
			},
			Severity: DiagnosticError,
			Message:  "Undefined rule 'Foo'",
			Code:     "undefined-rule",
			FilePath: "test.peg",
		},
		{
			Location: SourceLocation{
				Span: Span{
					Start: NewLocation(20, 1, 200),
					End:   NewLocation(20, 10, 210),
				},
			},
			Severity: DiagnosticWarning,
			Message:  "Unused rule 'Bar'",
			Code:     "unused-rule",
			FilePath: "test.peg",
		},
	}

	err := NewGrammarError(diagnostics)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ge, ok := err.(*GrammarError)
	if !ok {
		t.Fatalf("expected *GrammarError, got %T", err)
	}

	if !ge.HasErrors() {
		t.Error("expected HasErrors() to be true")
	}
	if ge.ErrorCount() != 1 {
		t.Errorf("expected 1 error, got %d", ge.ErrorCount())
	}
	if ge.WarningCount() != 1 {
		t.Errorf("expected 1 warning, got %d", ge.WarningCount())
	}

	errStr := ge.Error()
	if !strings.Contains(errStr, "test.peg:10:5") {
		t.Errorf("error string should contain location, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Undefined rule") {
		t.Errorf("error string should contain message, got: %s", errStr)
	}
}

func TestDiagnosticFormatCLI(t *testing.T) {
	d := Diagnostic{
		Location: SourceLocation{
			Span: Span{
				Start: NewLocation(10, 5, 100),
				End:   NewLocation(10, 15, 110),
			},
		},
		Severity: DiagnosticError,
		Message:  "Something went wrong",
		Code:     "test-error",
		FilePath: "/path/to/file.peg",
	}

	formatted := d.FormatCLI()
	expected := "/path/to/file.peg:10:5: error: Something went wrong [test-error]"
	if formatted != expected {
		t.Errorf("expected:\n  %s\ngot:\n  %s", expected, formatted)
	}
}

func TestQueryDiagnosticsAsError(t *testing.T) {
	tests := []struct {
		name       string
		grammar    string
		wantErr    bool
		wantNilErr bool
	}{
		{
			name:       "valid grammar",
			grammar:    `G <- "hello"`,
			wantNilErr: true,
		},
		{
			name:    "syntax error",
			grammar: `G <- "hello`,
			wantErr: true,
		},
		{
			name:    "undefined reference",
			grammar: `G <- Undefined`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))

			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			db := NewDatabase(cfg, loader)

			err := QueryDiagnosticsAsError(db, "test.peg")

			if tt.wantNilErr && err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseErrorsQuery_MultilineLocation(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Error on line 3
	loader.Add("multiline.peg", []byte(`
G <- A
A <- "hello
`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, ParseErrorsQuery, FilePath("multiline.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diagnostics) == 0 {
		t.Fatal("expected parse errors, got none")
	}

	d := diagnostics[0]
	// The error should be on line 3 where the unclosed string starts
	if d.Location.Span.Start.Line < 2 {
		t.Errorf("expected error on line >= 2, got line %d", d.Location.Span.Start.Line)
	}
}

func TestAllParseErrorsQuery_ImportedFileErrors(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Main file is valid
	loader.Add("main.peg", []byte(`
@import Helper from "./helper.peg"
G <- Helper
`))
	// Imported file has an error
	loader.Add("helper.peg", []byte(`Helper <- "hello`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, AllParseErrorsQuery, FilePath("main.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diagnostics) == 0 {
		t.Fatal("expected parse errors from imported file, got none")
	}

	// The error should be from helper.peg
	foundHelperError := false
	for _, d := range diagnostics {
		if d.FilePath == "helper.peg" {
			foundHelperError = true
			break
		}
	}
	if !foundHelperError {
		t.Errorf("expected error from helper.peg, got errors: %v", diagnostics)
	}
}

func TestDiagnosticsQuery_SemanticErrorInImport(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Main file imports helper
	loader.Add("main.peg", []byte(`
@import Helper from "./helper.peg"
G <- Helper UndefinedInMain
`))
	// Imported file is valid
	loader.Add("helper.peg", []byte(`Helper <- "hello"`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("main.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have undefined rule error from main.peg
	foundError := false
	for _, d := range diagnostics {
		if d.Code == "undefined-rule" && strings.Contains(d.Message, "UndefinedInMain") {
			foundError = true
			if d.FilePath != "main.peg" {
				t.Errorf("expected error to be in main.peg, got %s", d.FilePath)
			}
			break
		}
	}
	if !foundError {
		t.Errorf("expected undefined-rule error, got: %v", diagnostics)
	}
}

func TestDiagnosticsQuery_CorrectFilePathsForAllDiagnostics(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Main file with an unused rule
	loader.Add("main.peg", []byte(`
@import Helper from "./helper.peg"
G <- Helper
UnusedMain <- "unused"
`))
	// Imported file
	loader.Add("helper.peg", []byte(`Helper <- "hello"`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("main.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that diagnostics have correct file paths
	mainUnused := false
	for _, d := range diagnostics {
		if d.Code == "unused-rule" && strings.Contains(d.Message, "UnusedMain") {
			mainUnused = true
			if d.FilePath != "main.peg" {
				t.Errorf("expected UnusedMain warning to be from main.peg, got %s", d.FilePath)
			}
		}
	}

	if !mainUnused {
		t.Error("expected unused-rule warning for UnusedMain in main.peg")
	}
}

func TestDiagnosticsQuery_UndefinedInImportedFile(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Main file imports helper
	loader.Add("main.peg", []byte(`
@import Helper from "./helper.peg"
G <- Helper
`))
	// Imported file has undefined reference
	loader.Add("helper.peg", []byte(`
Helper <- Inner
Inner <- UndefinedInHelper
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("main.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have undefined rule error from helper.peg
	foundHelperError := false
	for _, d := range diagnostics {
		if d.Code == "undefined-rule" && strings.Contains(d.Message, "UndefinedInHelper") {
			foundHelperError = true
			if d.FilePath != "helper.peg" {
				t.Errorf("expected error from helper.peg, got %s", d.FilePath)
			}
			// Check line number is valid
			if d.Location.Span.Start.Line < 1 {
				t.Errorf("expected valid line number, got %d", d.Location.Span.Start.Line)
			}
		}
	}
	if !foundHelperError {
		t.Errorf("expected undefined-rule error for UndefinedInHelper from helper.peg, got: %v", diagnostics)
	}
}

func TestDiagnosticsQuery_MultipleErrorsAcrossFiles(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("main.peg", []byte(`
@import Helper from "./helper.peg"
G <- Helper UndefinedMain
`))
	loader.Add("helper.peg", []byte(`
Helper <- UndefinedHelper
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("main.peg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count errors by file
	mainErrors := 0
	helperErrors := 0
	for _, d := range diagnostics {
		if d.Code == "undefined-rule" {
			if d.FilePath == "main.peg" {
				mainErrors++
			} else if d.FilePath == "helper.peg" {
				helperErrors++
			}
		}
	}

	if mainErrors == 0 {
		t.Error("expected at least one undefined-rule error from main.peg")
	}
	if helperErrors == 0 {
		t.Error("expected at least one undefined-rule error from helper.peg")
	}
}
