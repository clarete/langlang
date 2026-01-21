package lsp

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize

type InitializeParams struct {
	// The process Id of the parent process that started the
	// server.  Is null if the process has not been started by
	// another process.  If the parent process is not alive then
	// the server should exit (see exit notification) its process
	ProcessID int `json:"processId,omitempty"`

	// The rootPath of the workspace. Is null if no folder is
	// open.
	//
	// @deprecated in favour of `rootUri`.
	RootPath string `json:"rootPath,omitempty"`

	// The rootUri of the workspace. Is null if no folder is
	// open. If both `rootPath` and `rootUri` are set `rootUri`
	// wins.
	//
	// @deprecated in favour of `workspaceFolders`
	RootURI string `json:"rootUri,omitempty"`

	// User provided initialization options
	InitializationOptions InitializeOptions `json:"initializationOptions,omitempty"`

	// The capabilities provided by the client (editor or tool)
	Capabilities ClientCapabilities `json:"capabilities,omitempty"`

	// The initial trace setting. If omitted trace is disabled ('off')
	Trace string `json:"trace,omitempty"`
}

type InitializeOptions struct{}

type InitializeResult struct {
	// The capabilities the language server provides.
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#clientCapabilities

type PositionEncodingKind string

const (
	UTF8  PositionEncodingKind = "utf-8"
	UTF16                      = "utf-16"
	UTF32                      = "utf-32"
)

type ClientCapabilities struct {
	PositionEncodings []PositionEncodingKind `json:"capabilities,omitempty"`
}

type TextDocumentSyncKind int

const (
	TDSKNone        TextDocumentSyncKind = 0
	TDSKFull        TextDocumentSyncKind = 1
	TDSKIncremental TextDocumentSyncKind = 2
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities

type ServerCapabilities struct {
	TextDocumentSync TextDocumentSyncKind `json:"textDocumentSync,omitempty"`
	HoverProvider    bool                 `json:"hoverProvider,omitempty"`

	// The server provides code actions. The `CodeActionOptions`
	// return type is only valid if the client signals code action
	// literal support via the property
	// `textDocument.codeAction.codeActionLiteralSupport`
	CompletionProvider *CompletionOptions `json:"completionProvider,omitempty"`

	// The server provides semantic tokens support.
	//
	// @since 3.16.0
	SemanticTokensProvider *SemanticTokensProvider `json:"semanticTokensProvider,omitempty"`

	// The server provides document symbol support
	DocumentSymbolProvider bool `json:"documentSymbolProvider,omitempty"`

	// SignatureHelpProvider            *SignatureHelpOptions            `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider bool `json:"definitionProvider,omitempty"`
	// TypeDefinitionProvider           bool                             `json:"typeDefinitionProvider,omitempty"`
	// ImplementationProvider           bool                             `json:"implementationProvider,omitempty"`
	// ReferencesProvider               bool                             `json:"referencesProvider,omitempty"`
	// DocumentHighlightProvider        bool                             `json:"documentHighlightProvider,omitempty"`
	// WorkspaceSymbolProvider          bool                             `json:"workspaceSymbolProvider,omitempty"`
	// CodeActionProvider               interface{}                      `json:"codeActionProvider,omitempty"`
	// CodeLensProvider                 *CodeLensOptions                 `json:"codeLensProvider,omitempty"`
	// DocumentFormattingProvider       bool                             `json:"documentFormattingProvider,omitempty"`
	// DocumentRangeFormattingProvider  bool                             `json:"documentRangeFormattingProvider,omitempty"`
	// DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
	// RenameProvider                   bool                             `json:"renameProvider,omitempty"`
	// DocumentLinkProvider             *DocumentLinkOptions             `json:"documentLinkProvider,omitempty"`
	// ColorProvider                    bool                             `json:"colorProvider,omitempty"`
	// FoldingRangeProvider             bool                             `json:"foldingRangeProvider,omitempty"`
	// DeclarationProvider              bool                             `json:"declarationProvider,omitempty"`
	// ExecuteCommandProvider           *ExecuteCommandOptions           `json:"executeCommandProvider,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didOpen

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didChange

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didSave

type DidSaveTextDocumentParams struct {
	Text         string                 `json:"text"`
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didClose

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#hoverParams

type HoverParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#hover

// Hover is the result of a hover request.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    Range         `json:"range,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#semanticTokensLegend

type SemanticTokensLegend struct {
	// The token types a server uses.
	TokenTypes []string `json:"tokenTypes,omitempty"`

	// The token modifiers a server uses.
	TokenModifiers []string `json:"tokenModifiers,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#semanticTokensOptions

type SemanticTokensOptions struct {
	// The legend used by the server
	Legend SemanticTokensLegend `json:"legend"`

	// Server supports providing semantic tokens for a specific range
	// of a document.
	Range bool `json:"range,omitempty"`

	// Server supports providing semantic tokens for a full
	// document.
	Full bool `json:"full,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#markupContent

type MarkupContent struct {
	// Kind The type of the Markup
	Kind MarkupKind `json:"kind"`

	// Value is the content itself
	Value string `json:"value"`
}

type MarkupKind string

const (
	// Plain text is supported as a content format
	MarkupKind_Plaintext MarkupKind = "plaintext"
	// Markdown is supported as a content format
	MarkupKind_Markdown = "markdown"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#semanticTokensRegistrationOptions

type SemanticTokensRegistrationOptions struct {
	// TextDocumentRegistrationOptions
	SemanticTokensOptions
	// StaticRegistrationOptions
}

type SemanticTokensProvider struct {
	SemanticTokensOptions
	// SemanticTokensRegistrationOptions
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionOptions

type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionParams

type CompletionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	Context *CompletionContext `json:"context,omitempty"`
}

type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

type CompletionTriggerKind int

const (
	CompletionTriggerKind_Invoked                         CompletionTriggerKind = 1
	CompletionTriggerKind_TriggerCharacter                CompletionTriggerKind = 2
	CompletionTriggerKind_TriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionList

// CompletionList Represents a collection of completion items to be
// presented in the editor.
type CompletionList struct {
	// This list is not complete. Further typing should result in
	// recomputing this list.
	//
	// Recomputed lists have all their items replaced (not
	// appended) in the incomplete completion sessions.
	IsIncomplete bool `json:"isIncomplete"`

	// Items carries the completion items
	Items []CompletionItem
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItem

type CompletionItem struct {
	Label         string              `json:"label"`
	Kind          CompletionItemKind  `json:"kind,omitempty"`
	Tags          []CompletionItemTag `json:"tags,omitempty"`
	Detail        string              `json:"detail,omitempty"`
	Documentation MarkupContent       `json:"documentation,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItemKind

type CompletionItemKind int

const (
	CompletionItemKind_Text          CompletionItemKind = 1
	CompletionItemKind_Method                           = 2
	CompletionItemKind_Function                         = 3
	CompletionItemKind_Constructor                      = 4
	CompletionItemKind_Field                            = 5
	CompletionItemKind_Variable                         = 6
	CompletionItemKind_Class                            = 7
	CompletionItemKind_Interface                        = 8
	CompletionItemKind_Module                           = 9
	CompletionItemKind_Property                         = 10
	CompletionItemKind_Unit                             = 11
	CompletionItemKind_Value                            = 12
	CompletionItemKind_Enum                             = 13
	CompletionItemKind_Keyword                          = 14
	CompletionItemKind_Snippet                          = 15
	CompletionItemKind_Color                            = 16
	CompletionItemKind_File                             = 17
	CompletionItemKind_Reference                        = 18
	CompletionItemKind_Folder                           = 19
	CompletionItemKind_EnumMember                       = 20
	CompletionItemKind_Constant                         = 21
	CompletionItemKind_Struct                           = 22
	CompletionItemKind_Event                            = 23
	CompletionItemKind_Operator                         = 24
	CompletionItemKind_TypeParameter                    = 25
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItemTag

type CompletionItemTag int

const (
	CompletionItemTag_Deprecated CompletionItemTag = 1
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#documentSymbolParams

type DocumentSymbolParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#symbolKind

type SymbolKind int

const (
	SymbolKind_File          SymbolKind = 1
	SymbolKind_Module                   = 2
	SymbolKind_Namespace                = 3
	SymbolKind_Package                  = 4
	SymbolKind_Class                    = 5
	SymbolKind_Method                   = 6
	SymbolKind_Property                 = 7
	SymbolKind_Field                    = 8
	SymbolKind_Constructor              = 9
	SymbolKind_Enum                     = 10
	SymbolKind_Interface                = 11
	SymbolKind_Function                 = 12
	SymbolKind_Variable                 = 13
	SymbolKind_Constant                 = 14
	SymbolKind_String                   = 15
	SymbolKind_Number                   = 16
	SymbolKind_Boolean                  = 17
	SymbolKind_Array                    = 18
	SymbolKind_Object                   = 19
	SymbolKind_Key                      = 20
	SymbolKind_Null                     = 21
	SymbolKind_EnumMember               = 22
	SymbolKind_Struct                   = 23
	SymbolKind_Event                    = 24
	SymbolKind_Operator                 = 25
	SymbolKind_TypeParameter            = 26
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#symbolTag

type SymbolTag int

const (
	SymbolTag_Deprecated SymbolTag = 1
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#documentSymbol

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind,omitempty"`
	Tags           []SymbolTag      `json:"tags,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// other important types

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentContentChangeEvent struct {
	Range       Range  `json:"range"`
	RangeLength int    `json:"rangeLength"`
	Text        string `json:"text"`
}
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type WorkDoneProgressParams struct {
	// type ProgressToken = integer | string
	WorkDoneToken interface{} `json:"workDoneToken,omitempty"`
}

// ---- Diagnostics ----
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#diagnostic
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#diagnosticSeverity
type DiagnosticSeverity int

const (
	DiagnosticSeverity_Error       DiagnosticSeverity = 1
	DiagnosticSeverity_Warning     DiagnosticSeverity = 2
	DiagnosticSeverity_Information DiagnosticSeverity = 3
	DiagnosticSeverity_Hint        DiagnosticSeverity = 4
)

// ---- Definition ----
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition

type DefinitionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#location
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}
