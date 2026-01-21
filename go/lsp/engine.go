package lsp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	langlang "github.com/clarete/langlang/go"
)

// Engine is the transport-independent core of the LSP implementation.
//
// It is intentionally independent from the transport so we can use it
// both within the IDE process where messages are exchanged in memory
// and also as its own standalone service, exchanging messages through
// the network.
type Engine struct {
	loader *LSPImportLoader
	db     *langlang.Database
	config *langlang.Config

	// open tracks the set of open documents (LSP
	// didOpen/didClose). We use this to recompute diagnostics for
	// all open docs on each change so import-related diagnostics
	// don't get stuck due to open order.
	open map[string]bool
}

// LSPImportLoader is an ImportLoader that supports LSP's open/close
// document semantics with in-memory overlays over a base loader.
type LSPImportLoader struct {
	base     langlang.ImportLoader
	overlays map[string][]byte
}

func NewLSPImportLoader(base langlang.ImportLoader) *LSPImportLoader {
	return &LSPImportLoader{
		base:     base,
		overlays: make(map[string][]byte),
	}
}

func (l *LSPImportLoader) SetOverlay(path string, content []byte) {
	l.overlays[path] = content
}

func (l *LSPImportLoader) ClearOverlay(path string) {
	delete(l.overlays, path)
}

func (l *LSPImportLoader) GetPath(importPath, parentPath string) (string, error) {
	return l.base.GetPath(importPath, parentPath)
}

func (l *LSPImportLoader) GetContent(path string) ([]byte, error) {
	if content, ok := l.overlays[path]; ok {
		return content, nil
	}
	return l.base.GetContent(path)
}

func NewEngine(base langlang.ImportLoader) *Engine {
	loader := NewLSPImportLoader(base)
	config := langlang.NewConfig()
	config.SetBool("grammar.add_builtins", true)
	return &Engine{
		loader: loader,
		db:     langlang.NewDatabase(config, loader),
		config: config,
		open:   map[string]bool{},
	}
}

func (e *Engine) Initialize(_ InitializeParams) InitializeResult {
	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:       TDSKFull,
			HoverProvider:          true,
			DocumentSymbolProvider: true,
			DefinitionProvider:     true,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			SemanticTokensProvider: &SemanticTokensProvider{},
		},
	}
}

func (e *Engine) DidOpen(params DidOpenTextDocumentParams) ([]PublishDiagnosticsParams, error) {
	e.open[params.TextDocument.URI] = true
	e.loader.SetOverlay(params.TextDocument.URI, []byte(params.TextDocument.Text))
	e.db.InvalidateFile(params.TextDocument.URI)
	return e.PublishDiagnosticsAllOpen(params.TextDocument.URI)
}

func (e *Engine) DidChange(params DidChangeTextDocumentParams) ([]PublishDiagnosticsParams, error) {
	if len(params.ContentChanges) != 1 {
		// full document sync only for now (no chunks)
		return []PublishDiagnosticsParams{{URI: params.TextDocument.URI, Diagnostics: []Diagnostic{}}}, nil
	}
	uri := params.TextDocument.URI
	e.open[uri] = true
	e.loader.SetOverlay(uri, []byte(params.ContentChanges[0].Text))
	e.db.InvalidateFile(uri)
	return e.PublishDiagnosticsForFile(uri)
}

func (e *Engine) DidClose(params DidCloseTextDocumentParams) ([]PublishDiagnosticsParams, error) {
	uri := params.TextDocument.URI
	delete(e.open, uri)
	e.loader.ClearOverlay(uri)
	e.db.InvalidateFile(uri)

	// Recompute for remaining open files so import/name
	// diagnostics update, plus explicitly clear the closed
	// document diagnostics.
	pubs, err := e.PublishDiagnosticsAllOpen("")
	pubs = append(pubs, PublishDiagnosticsParams{URI: uri, Diagnostics: []Diagnostic{}})
	return pubs, err
}

func (e *Engine) PublishDiagnosticsAllOpen(_ string) ([]PublishDiagnosticsParams, error) {
	merged := map[string][]langlang.Diagnostic{}
	for uri := range e.open {
		diagnostics, err := langlang.QueryDiagnostics(e.db, uri)
		if err != nil {
			// best-effort: keep going, but surface error on that root
			merged[uri] = append(merged[uri], langlang.Diagnostic{
				Location: langlang.SourceLocation{
					FileID: -1,
					Span:   langlang.Span{Start: langlang.NewLocation(1, 1, 0), End: langlang.NewLocation(1, 1, 0)},
				},
				Severity: langlang.DiagnosticError,
				Code:     "check_failed",
				Message:  err.Error(),
				FilePath: uri,
			})
			continue
		}
		for _, d := range diagnostics {
			filePath := d.FilePath
			if filePath == "" {
				filePath = uri
			}
			merged[filePath] = append(merged[filePath], d)
		}
	}

	// Ensure every open document gets a publish even if it has no
	// diagnostics, so stale diagnostics are cleared.
	for uri := range e.open {
		if _, ok := merged[uri]; !ok {
			merged[uri] = nil
		}
	}

	out := make([]PublishDiagnosticsParams, 0, len(merged))
	for uri, diags := range merged {
		ds := make([]Diagnostic, 0, len(diags))
		seen := map[string]bool{}
		for _, d := range diags {
			key := fmt.Sprintf("%s|%s|%s|%s", d.Code, d.Message, d.FilePath, d.Location.Span.String())
			if seen[key] {
				continue
			}
			seen[key] = true
			ds = append(ds, Diagnostic{
				Range:    toLspRange(d.Location.Span),
				Severity: toLspSeverity(d.Severity),
				Source:   "langlang",
				Message:  d.Message,
			})
		}
		out = append(out, PublishDiagnosticsParams{URI: uri, Diagnostics: ds})
	}
	return out, nil
}

// PublishDiagnosticsForFile computes diagnostics for a single file.
// This is used during typing (DidChange) for better responsiveness -
// we don't need to recompute diagnostics for ALL open files on every
// keystroke.
func (e *Engine) PublishDiagnosticsForFile(uri string) ([]PublishDiagnosticsParams, error) {
	diagnostics, err := langlang.QueryDiagnostics(e.db, uri)
	if err != nil {
		// Return error as a diagnostic so the user sees it
		return []PublishDiagnosticsParams{{
			URI: uri,
			Diagnostics: []Diagnostic{{
				Range:    Range{Start: Position{}, End: Position{}},
				Severity: DiagnosticSeverity_Error,
				Source:   "langlang",
				Message:  err.Error(),
			}},
		}}, nil
	}

	// Group diagnostics by file (the changed file may report
	// diagnostics in imported files too)
	diagsByURI := make(map[string][]Diagnostic)
	for _, d := range diagnostics {
		filePath := d.FilePath
		if filePath == "" {
			filePath = uri
		}
		diagsByURI[filePath] = append(diagsByURI[filePath], Diagnostic{
			Range:    toLspRange(d.Location.Span),
			Severity: toLspSeverity(d.Severity),
			Source:   "langlang",
			Message:  d.Message,
		})
	}

	// Ensure the requested file always gets a publish (even if empty)
	// so stale diagnostics are cleared
	if _, ok := diagsByURI[uri]; !ok {
		diagsByURI[uri] = nil
	}

	out := make([]PublishDiagnosticsParams, 0, len(diagsByURI))
	for fileURI, diags := range diagsByURI {
		out = append(out, PublishDiagnosticsParams{URI: fileURI, Diagnostics: diags})
	}
	return out, nil
}

func (e *Engine) DocumentSymbol(params DocumentSymbolParams) ([]DocumentSymbol, error) {
	uri := params.TextDocument.URI

	// Use ParsedGrammarQuery to get only symbols from this file
	// (not imports or builtins)
	grammar, err := langlang.Get(e.db, langlang.ParsedGrammarQuery, langlang.FilePath(uri))
	if err != nil {
		return nil, err
	}

	out := make([]DocumentSymbol, 0, len(grammar.Definitions))
	for _, def := range grammar.Definitions {
		rng := toLspRange(def.SourceLocation().Span)
		out = append(out, DocumentSymbol{
			Name:           def.Name,
			Detail:         "production",
			Kind:           SymbolKind_Function,
			Range:          rng,
			SelectionRange: rng,
		})
	}
	return out, nil
}

func (e *Engine) Definition(params DefinitionParams) ([]Location, error) {
	uri := params.TextDocument.URI

	// Convert LSP position to cursor offset
	cursor, err := e.positionToCursor(uri, params.Position)
	if err != nil {
		return nil, nil
	}

	// Check if we're on an import path or name first
	if impURI, ok := e.importURIAtPosition(uri, params.Position.Line, params.Position.Character); ok {
		zero := Range{Start: Position{}, End: Position{}}
		return []Location{{URI: impURI, Range: zero}}, nil
	}

	// Use the SymbolAtPosition query
	symbol, err := langlang.Get(e.db, langlang.SymbolAtCursorQuery, langlang.CursorKey{
		File:   uri,
		Cursor: cursor,
	})
	if err != nil || symbol == nil {
		return nil, nil
	}

	if symbol.DefinitionLoc != nil {
		defURI := e.db.FileIDToPath(symbol.DefinitionLoc.FileID)
		if defURI == "" {
			defURI = uri
		}
		return []Location{{
			URI:   defURI,
			Range: toLspRange(symbol.DefinitionLoc.Span),
		}}, nil
	}

	return nil, nil
}

func (e *Engine) Hover(params HoverParams) (*Hover, error) {
	uri := params.TextDocument.URI

	// Check for import hover first
	if impURI, impSpan, ok := e.importURIAndSpanAtPosition(uri, params.Position.Line, params.Position.Character); ok {
		display := displayURI(impURI)
		return &Hover{
			Range: toLspRange(impSpan),
			Contents: MarkupContent{
				Kind:  MarkupKind_Markdown,
				Value: fmt.Sprintf("**Imported from** `%s`", display),
			},
		}, nil
	}

	// Convert LSP position to cursor offset
	cursor, err := e.positionToCursor(uri, params.Position)
	if err != nil {
		return nil, nil
	}

	// Use the HoverInfo query
	hover, err := langlang.Get(e.db, langlang.HoverInfoQuery, langlang.CursorKey{
		File:   uri,
		Cursor: cursor,
	})
	if err != nil || hover == nil {
		return nil, nil
	}

	return &Hover{
		Range: toLspRange(hover.Range.Span),
		Contents: MarkupContent{
			Kind:  MarkupKind_Markdown,
			Value: hover.Contents,
		},
	}, nil
}

func (e *Engine) Completion(params CompletionParams) ([]CompletionItem, error) {
	uri := params.TextDocument.URI

	// Convert LSP position to cursor offset
	cursor, err := e.positionToCursor(uri, params.Position)
	if err != nil {
		return nil, nil
	}

	// Use the CompletionItems query
	items, err := langlang.Get(e.db, langlang.CompletionItemsQuery, langlang.CursorKey{
		File:   uri,
		Cursor: cursor,
	})
	if err != nil || items == nil {
		return nil, nil
	}

	// Convert to LSP CompletionItems
	out := make([]CompletionItem, 0, len(items))
	for _, item := range items {
		out = append(out, CompletionItem{
			Label:  item.Label,
			Kind:   toLspCompletionKind(item.Kind),
			Detail: item.Detail,
			Documentation: MarkupContent{
				Kind:  MarkupKind_Plaintext,
				Value: item.Documentation,
			},
		})
	}
	return out, nil
}

func toLspCompletionKind(k langlang.CompletionKind) CompletionItemKind {
	switch k {
	case langlang.CompletionKindRule:
		return CompletionItemKind_Function
	case langlang.CompletionKindKeyword:
		return CompletionItemKind_Keyword
	case langlang.CompletionKindSnippet:
		return CompletionItemKind_Snippet
	case langlang.CompletionKindLabel:
		return CompletionItemKind_EnumMember
	default:
		return CompletionItemKind_Text
	}
}

func (e *Engine) positionToCursor(uri string, pos Position) (int, error) {
	return langlang.Get(e.db, langlang.CursorAtLocationQuery, langlang.LocationKey{
		File:   uri,
		Line:   pos.Line,
		Column: pos.Character,
	})
}

func (e *Engine) importURIAtPosition(parentURI string, line0, char0 int) (string, bool) {
	uri, _, ok := e.importURIAndSpanAtPosition(parentURI, line0, char0)
	return uri, ok
}

func (e *Engine) importURIAndSpanAtPosition(
	parentURI string,
	line0, char0 int,
) (string, langlang.Span, bool) {
	line := line0 + 1
	col := char0 + 1

	grammar, err := langlang.Get(e.db, langlang.ParsedGrammarQuery, langlang.FilePath(parentURI))
	if err != nil || grammar == nil {
		return "", langlang.Span{}, false
	}

	for _, imp := range grammar.Imports {
		impLoc := imp.SourceLocation()
		if !posInSpan(line, col, impLoc.Span) {
			continue
		}

		// Resolve the import path
		childPath, err := e.loader.GetPath(imp.GetPath(), parentURI)
		if err != nil {
			continue
		}

		return childPath, impLoc.Span, true
	}

	return "", langlang.Span{}, false
}

func displayURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u == nil {
		return uri
	}
	if u.Scheme == "inmemory" {
		// e.g. inmemory://project/foo/bar.peg -> foo/bar.peg
		return strings.TrimPrefix(u.Path, "/")
	}
	if u.Scheme == "file" {
		return u.Path
	}
	if u.Path != "" {
		return strings.TrimPrefix(u.Path, "/")
	}
	return uri
}

func posInSpan(line, col int, sp langlang.Span) bool {
	if line < sp.Start.Line || line > sp.End.Line {
		return false
	}
	if line == sp.Start.Line && col < sp.Start.Column {
		return false
	}
	if line == sp.End.Line && col > sp.End.Column {
		return false
	}
	return true
}

func toLspRange(s langlang.Span) Range {
	return Range{
		Start: Position{
			Line:      max(s.Start.Line - 1),
			Character: max(s.Start.Column - 1),
		},
		End: Position{
			Line:      max(s.End.Line - 1),
			Character: max(s.End.Column - 1),
		},
	}
}

func toLspSeverity(s langlang.DiagnosticSeverity) DiagnosticSeverity {
	switch s {
	case langlang.DiagnosticError:
		return DiagnosticSeverity_Error
	case langlang.DiagnosticWarning:
		return DiagnosticSeverity_Warning
	case langlang.DiagnosticInfo:
		return DiagnosticSeverity_Information
	default:
		return DiagnosticSeverity_Error
	}
}

// HandleJSONRPC allows embedding the LSP engine in environments
// without a JSON-RPC transport (e.g. browser WASM). It accepts a
// single JSON-RPC message (as a string) and returns a JSON array
// (string) containing all outgoing messages generated while handling
// it (responses and/or notifications).
func (e *Engine) HandleJSONRPC(input string) (string, error) {
	var msg rpcMessage
	if err := json.Unmarshal([]byte(input), &msg); err != nil {
		return "", err
	}
	if msg.JSONRPC == "" {
		msg.JSONRPC = "2.0"
	}

	var out []rpcMessage
	respond := func(result any) {
		if len(msg.ID) == 0 {
			return
		}
		out = append(out, rpcMessage{JSONRPC: "2.0", ID: msg.ID, Result: result})
	}
	notify := func(method string, params any) {
		out = append(out, rpcMessage{
			JSONRPC: "2.0",
			Method:  method,
			Params:  mustJSON(params),
		})
	}

	switch msg.Method {
	case "initialize":
		var p InitializeParams
		_ = json.Unmarshal(msg.Params, &p)
		respond(e.Initialize(p))
	case "initialized":
		// no-op
	case "textDocument/didOpen":
		var p DidOpenTextDocumentParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		pubs, _ := e.DidOpen(p)
		for _, pub := range pubs {
			notify("textDocument/publishDiagnostics", pub)
		}
	case "textDocument/didChange":
		var p DidChangeTextDocumentParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		pubs, _ := e.DidChange(p)
		for _, pub := range pubs {
			notify("textDocument/publishDiagnostics", pub)
		}
	case "textDocument/didClose":
		var p DidCloseTextDocumentParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		pubs, _ := e.DidClose(p)
		for _, pub := range pubs {
			notify("textDocument/publishDiagnostics", pub)
		}
	case "textDocument/definition":
		var p DefinitionParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		locs, _ := e.Definition(p)
		respond(locs)
	case "textDocument/hover":
		var p HoverParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		h, _ := e.Hover(p)
		respond(h)
	case "textDocument/documentSymbol":
		var p DocumentSymbolParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		syms, _ := e.DocumentSymbol(p)
		respond(syms)
	case "textDocument/completion":
		var p CompletionParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return "", err
		}
		items, _ := e.Completion(p)
		respond(items)
	default:
		respond(nil)
	}

	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
