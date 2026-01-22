package langlang

import (
	"strings"
	"testing"
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
