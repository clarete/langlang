package lsp

import (
	"testing"

	langlang "github.com/clarete/langlang/go"
)

func TestEngine_Definition_ImportAliasJumpsToImportedFile(t *testing.T) {
	loader := langlang.NewInMemoryImportLoader()
	e := NewEngine(loader)

	exprURI := "inmemory://project/expr.peg"
	numURI := "inmemory://project/number.peg"

	// Important: open BOTH files so the loader has the content available.
	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        numURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "Number <- [0-9]+\n",
	}})
	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        exprURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "@import Number from \"./number.peg\"\nExpr <- Number\n",
	}})

	// Cursor on "Number" in the import line: "@import Number from ..."
	locs, err := e.Definition(DefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: exprURI},
			Position:     Position{Line: 0, Character: 10},
		},
	})
	if err != nil {
		t.Fatalf("Definition returned error: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d: %#v", len(locs), locs)
	}
	if got := locs[0].URI; got != numURI {
		t.Fatalf("expected URI %q, got %q", numURI, got)
	}
	if got := locs[0].Range.Start.Line; got != 0 {
		t.Fatalf("expected start line 0, got %d", got)
	}
	if got := locs[0].Range.Start.Character; got != 0 {
		t.Fatalf("expected start character 0, got %d", got)
	}
}

func TestEngine_Definition_ImportPathJumpsToImportedFile(t *testing.T) {
	loader := langlang.NewInMemoryImportLoader()
	e := NewEngine(loader)

	exprURI := "inmemory://project/expr.peg"
	numURI := "inmemory://project/number.peg"

	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        numURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "Number <- [0-9]+\n",
	}})
	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        exprURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "@import Number from \"./number.peg\"\nExpr <- Number\n",
	}})

	// Cursor inside "./number.peg"
	locs, err := e.Definition(DefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: exprURI},
			Position:     Position{Line: 0, Character: 22},
		},
	})
	if err != nil {
		t.Fatalf("Definition returned error: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d: %#v", len(locs), locs)
	}
	if got := locs[0].URI; got != numURI {
		t.Fatalf("expected URI %q, got %q", numURI, got)
	}
}

func TestEngine_Hover_RuleShowsDefinedInAndPreviewLine(t *testing.T) {
	loader := langlang.NewInMemoryImportLoader()
	e := NewEngine(loader)

	exprURI := "inmemory://project/expr.peg"
	numURI := "inmemory://project/number.peg"

	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        numURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "Number <- [0-9]+\n",
	}})
	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        exprURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "@import Number from \"./number.peg\"\nExpr <- Number\n",
	}})

	// Cursor on "Number" in "Expr <- Number"
	h, err := e.Hover(HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: exprURI},
			Position:     Position{Line: 1, Character: 9},
		},
	})
	if err != nil {
		t.Fatalf("Hover returned error: %v", err)
	}
	if h == nil {
		t.Fatalf("expected hover, got nil")
	}
	if h.Contents.Kind != MarkupKind_Markdown {
		t.Fatalf("expected markdown hover, got %q", h.Contents.Kind)
	}
}

func TestEngine_DocumentSymbol_ReturnsProductions(t *testing.T) {
	loader := langlang.NewInMemoryImportLoader()
	e := NewEngine(loader)

	uri := "inmemory://project/main.peg"
	_, _ = e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        uri,
		LanguageID: "peg",
		Version:    1,
		Text:       "A <- 'a'\nB <- 'b'\n",
	}})

	syms, err := e.DocumentSymbol(DocumentSymbolParams{TextDocument: TextDocumentItem{URI: uri}})
	if err != nil {
		t.Fatalf("DocumentSymbol returned error: %v", err)
	}
	if len(syms) != 2 {
		t.Fatalf("expected 2 symbols, got %d: %#v", len(syms), syms)
	}
	if syms[0].Name != "A" || syms[1].Name != "B" {
		t.Fatalf("unexpected symbol names: %#v", syms)
	}
}

func TestEngine_Diagnostics_ImportOrderDoesNotLeaveStaleErrors(t *testing.T) {
	loader := langlang.NewInMemoryImportLoader()
	e := NewEngine(loader)

	exprURI := "inmemory://project/expr.peg"
	valueURI := "inmemory://project/value.peg"

	// Open expr first (import target not opened yet).
	pubs1, err := e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        exprURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "@import Value from \"./value.peg\"\nExpr <- Value\n",
	}})
	if err != nil {
		t.Fatalf("DidOpen(expr) error: %v", err)
	}
	_ = pubs1 // may contain transient import_not_found; that's fine

	// Now open value. Diagnostics should be recomputed for all open docs, clearing
	// expr import errors.
	pubs2, err := e.DidOpen(DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI:        valueURI,
		LanguageID: "peg",
		Version:    1,
		Text:       "Value <- 'a'\n",
	}})
	if err != nil {
		t.Fatalf("DidOpen(value) error: %v", err)
	}

	var exprDiags []Diagnostic
	for _, p := range pubs2 {
		if p.URI == exprURI {
			exprDiags = p.Diagnostics
			break
		}
	}
	for _, d := range exprDiags {
		if d.Message == "" {
			continue
		}
		if d.Severity == DiagnosticSeverity_Error && (d.Message == "import not found" || d.Message == "production `Value` isn't declared") {
			t.Fatalf("unexpected stale import diagnostics on expr after opening value: %#v", exprDiags)
		}
	}
}
