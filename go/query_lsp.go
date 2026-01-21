package langlang

import (
	"fmt"
	"sort"
	"strings"
)

// SymbolKind enumerates the different kinds of symbols in a grammar.
type SymbolKind int

const (
	SymbolKindDefinition SymbolKind = iota // G <- ...
	SymbolKindIdentifier                   // reference to a rule
	SymbolKindLabel                        // ^LabelName (error label)
	SymbolKindLiteral                      // "hello"
	SymbolKindClass                        // [a-z]
	SymbolKindOperator                     // *, +, ?, &, !, /
)

func (k SymbolKind) String() string {
	switch k {
	case SymbolKindDefinition:
		return "Definition"
	case SymbolKindIdentifier:
		return "Identifier"
	case SymbolKindLabel:
		return "Label"
	case SymbolKindLiteral:
		return "Literal"
	case SymbolKindClass:
		return "Class"
	case SymbolKindOperator:
		return "Operator"
	default:
		return "Unknown"
	}
}

// SymbolInfo contains information about a symbol at a position.
type SymbolInfo struct {
	Name          string
	Kind          SymbolKind
	Location      SourceLocation
	DefinitionLoc *SourceLocation // where it's defined (for identifiers/labels)
}

// LabelLocation represents a label throw site (expr^Label).
type LabelLocation struct {
	Label    string
	Location SourceLocation
}

// RecoveryRuleInfo contains information about a label and its recovery rule.
type RecoveryRuleInfo struct {
	LabelName     string
	DefinitionLoc *SourceLocation  // where the recovery rule is defined (nil if missing)
	UsageLocs     []SourceLocation // all places where ^Label is used
	HasRecovery   bool             // true if a matching rule exists
}

// DocumentSymbol represents a symbol in the document outline.
type DocumentSymbol struct {
	Name     string
	Kind     SymbolKind
	Location SourceLocation
	Detail   string           // additional info (e.g., the rule's expression summary)
	Children []DocumentSymbol // for nested structure
}

// SemanticTokenType discriminates token types for semantic highlighting.
type SemanticTokenType int

const (
	SemanticTokenTypeDefinition SemanticTokenType = iota
	SemanticTokenTypeIdentifier
	SemanticTokenTypeLabel
	SemanticTokenTypeLiteral
	SemanticTokenTypeClass
	SemanticTokenTypeOperator
	SemanticTokenTypeComment
	SemanticTokenTypeKeyword // for @import, from
)

func (t SemanticTokenType) String() string {
	switch t {
	case SemanticTokenTypeDefinition:
		return "definition"
	case SemanticTokenTypeIdentifier:
		return "identifier"
	case SemanticTokenTypeLabel:
		return "label"
	case SemanticTokenTypeLiteral:
		return "string"
	case SemanticTokenTypeClass:
		return "class"
	case SemanticTokenTypeOperator:
		return "operator"
	case SemanticTokenTypeComment:
		return "comment"
	case SemanticTokenTypeKeyword:
		return "keyword"
	default:
		return "unknown"
	}
}

// SemanticToken represents a token for semantic highlighting.
type SemanticToken struct {
	Location  SourceLocation
	TokenType SemanticTokenType
	Modifiers []string // e.g., "recursive", "unused", "unresolved"
}

// HoverInfo contains information to display on hover.
type HoverInfo struct {
	Contents string // Markdown content
	Range    SourceLocation
}

// CompletionItem represents a completion suggestion.
type CompletionItem struct {
	Label         string
	Kind          CompletionKind
	Detail        string // e.g., "G <- ..."
	Documentation string
	InsertText    string
}

// CompletionKind discriminates completion item types.
type CompletionKind int

const (
	CompletionKindRule CompletionKind = iota
	CompletionKindKeyword
	CompletionKindSnippet
	CompletionKindLabel
)

// PositionKey is a query key for position-based queries.
type PositionKey struct {
	File   string
	Cursor int // byte offset
}

// ReferencesKey is a query key for finding references to a symbol.
type ReferencesKey struct {
	File       string
	SymbolName string
}

// Symbol Queries

// DefinitionLocationsQuery maps definition names to their source
// locations.  Used for: Go to Definition, Document Symbols
var DefinitionLocationsQuery = &Query[FilePath, map[string]SourceLocation]{
	Name:    "DefinitionLocations",
	Compute: computeDefinitionLocations,
}

func computeDefinitionLocations(db *Database, key FilePath) (map[string]SourceLocation, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}

	result := make(map[string]SourceLocation, len(grammar.Definitions))
	for _, def := range grammar.Definitions {
		result[def.Name] = def.SourceLocation()
	}
	return result, nil
}

// IdentifierLocationsQuery finds all identifier references in a file.
// Used for: Find References, Rename, Semantic Tokens
var IdentifierLocationsQuery = &Query[FilePath, []IdentifierLocation]{
	Name:    "IdentifierLocations",
	Compute: computeIdentifierLocations,
}

func computeIdentifierLocations(db *Database, key FilePath) ([]IdentifierLocation, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}

	var locations []IdentifierLocation

	for _, def := range grammar.Definitions {
		// Add the definition itself
		locations = append(locations, IdentifierLocation{
			Name:         def.Name,
			Location:     def.SourceLocation(),
			IsDefinition: true,
		})

		// Find all identifier references in the expression
		Inspect(def.Expr, func(n AstNode) bool {
			if id, ok := n.(*IdentifierNode); ok {
				locations = append(locations, IdentifierLocation{
					Name:         id.Value,
					Location:     id.SourceLocation(),
					IsDefinition: false,
				})
			}
			return true
		})
	}

	return locations, nil
}

// LabelLocationsQuery finds all label throw sites (expr^Label) in a
// file.  Used for: Find References to recovery rules, semantic tokens
var LabelLocationsQuery = &Query[FilePath, []LabelLocation]{
	Name:    "LabelLocations",
	Compute: computeLabelLocations,
}

func computeLabelLocations(db *Database, key FilePath) ([]LabelLocation, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}

	var labels []LabelLocation
	for _, def := range grammar.Definitions {
		Inspect(def.Expr, func(n AstNode) bool {
			if labeled, ok := n.(*LabeledNode); ok {
				labels = append(labels, LabelLocation{
					Label:    labeled.Label,
					Location: labeled.SourceLocation(),
				})
			}
			return true
		})
	}
	return labels, nil
}

// RecoveryRulesQuery maps label names to their recovery rule
// definitions.  A recovery rule is a definition whose name matches a
// label used in the grammar.  Used for: Go to Definition on labels,
// diagnostics for missing recovery rules
var RecoveryRulesQuery = &Query[FilePath, map[string]*RecoveryRuleInfo]{
	Name:    "RecoveryRules",
	Compute: computeRecoveryRules,
}

func computeRecoveryRules(db *Database, key FilePath) (map[string]*RecoveryRuleInfo, error) {
	labels, err := Get(db, LabelLocationsQuery, key)
	if err != nil {
		return nil, err
	}

	defLocs, err := Get(db, DefinitionLocationsQuery, key)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*RecoveryRuleInfo)

	// Group label usages
	for _, label := range labels {
		if info, ok := result[label.Label]; ok {
			info.UsageLocs = append(info.UsageLocs, label.Location)
		} else {
			result[label.Label] = &RecoveryRuleInfo{
				LabelName: label.Label,
				UsageLocs: []SourceLocation{label.Location},
			}
		}
	}

	// Link to recovery rule definitions
	for labelName, info := range result {
		if defLoc, ok := defLocs[labelName]; ok {
			info.DefinitionLoc = &defLoc
			info.HasRecovery = true
		}
	}

	return result, nil
}

// DocumentSymbolsQuery returns the symbol tree for outline view.
// Used for: textDocument/documentSymbol, breadcrumbs
var DocumentSymbolsQuery = &Query[FilePath, []DocumentSymbol]{
	Name:    "DocumentSymbols",
	Compute: computeDocumentSymbols,
}

func computeDocumentSymbols(db *Database, key FilePath) ([]DocumentSymbol, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}

	recursiveSet, err := Get(db, RecursiveSetQuery, key)
	if err != nil {
		return nil, err
	}

	labels, err := Get(db, LabelLocationsQuery, key)
	if err != nil {
		return nil, err
	}

	var symbols []DocumentSymbol

	// Add definitions
	for _, def := range grammar.Definitions {
		detail := summarizeExpression(def.Expr)
		if _, isRecursive := recursiveSet[def.Name]; isRecursive {
			detail = "(recursive) " + detail
		}

		symbols = append(symbols, DocumentSymbol{
			Name:     def.Name,
			Kind:     SymbolKindDefinition,
			Location: def.SourceLocation(),
			Detail:   detail,
		})
	}

	// Add unique labels as symbols
	seenLabels := make(map[string]bool)
	for _, label := range labels {
		if !seenLabels[label.Label] {
			seenLabels[label.Label] = true
			symbols = append(symbols, DocumentSymbol{
				Name:     "^" + label.Label,
				Kind:     SymbolKindLabel,
				Location: label.Location,
				Detail:   "error label",
			})
		}
	}

	return symbols, nil
}

// summarizeExpression creates a short string representation of an
// expression.
func summarizeExpression(expr AstNode) string {
	if expr == nil {
		return ""
	}
	s := expr.String()
	if len(s) > 50 {
		return s[:47] + "..."
	}
	return s
}

// Position-based Queries

// SymbolAtPositionQuery returns the symbol at a given cursor
// position.  Used for: Go to Definition, Hover, Find References
var SymbolAtPositionQuery = &Query[PositionKey, *SymbolInfo]{
	Name:    "SymbolAtPosition",
	Compute: computeSymbolAtPosition,
}

func computeSymbolAtPosition(db *Database, key PositionKey) (*SymbolInfo, error) {
	grammar, err := Get(db, ResolvedImportsQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}

	defLocs, err := Get(db, DefinitionLocationsQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}

	var result *SymbolInfo
	var resultSpecificity int // smaller span = more specific

	for _, def := range grammar.Definitions {
		// Check if cursor is on the definition name
		defLoc := def.SourceLocation()
		if containsCursor(defLoc, key.Cursor) {
			// Check if we're specifically on the name
			// part (the name comes before " <- ")
			nameEnd := defLoc.Span.Start.Cursor + len(def.Name)
			if key.Cursor <= nameEnd {
				specificity := len(def.Name)
				if result == nil || specificity < resultSpecificity {
					result = &SymbolInfo{
						Name:          def.Name,
						Kind:          SymbolKindDefinition,
						Location:      defLoc,
						DefinitionLoc: &defLoc,
					}
					resultSpecificity = specificity
				}
			}
		}

		// Search in the expression
		Inspect(def.Expr, func(n AstNode) bool {
			if n == nil {
				return true
			}
			loc := n.SourceLocation()
			if !containsCursor(loc, key.Cursor) {
				return true
			}

			specificity := loc.Span.End.Cursor - loc.Span.Start.Cursor
			if result != nil && specificity >= resultSpecificity {
				return true // Keep more specific result
			}

			switch node := n.(type) {
			case *IdentifierNode:
				info := &SymbolInfo{
					Name:     node.Value,
					Kind:     SymbolKindIdentifier,
					Location: loc,
				}
				if defLoc, ok := defLocs[node.Value]; ok {
					info.DefinitionLoc = &defLoc
				}
				result = info
				resultSpecificity = specificity

			case *LabeledNode:
				// For labeled nodes, check if cursor
				// is on the label part The label is
				// after the ^ character
				exprEnd := node.Expr.SourceLocation().Span.End.Cursor
				if key.Cursor > exprEnd {
					info := &SymbolInfo{
						Name:     node.Label,
						Kind:     SymbolKindLabel,
						Location: loc,
					}
					// Link to recovery rule if it exists
					if defLoc, ok := defLocs[node.Label]; ok {
						info.DefinitionLoc = &defLoc
					}
					result = info
					resultSpecificity = specificity
				}

			case *LiteralNode:
				result = &SymbolInfo{
					Name:     node.Value,
					Kind:     SymbolKindLiteral,
					Location: loc,
				}
				resultSpecificity = specificity

			case *ClassNode, *CharsetNode, *RangeNode:
				result = &SymbolInfo{
					Name:     n.String(),
					Kind:     SymbolKindClass,
					Location: loc,
				}
				resultSpecificity = specificity
			}

			return true
		})
	}
	return result, nil
}

// containsCursor checks if a source location contains the given cursor position.
func containsCursor(loc SourceLocation, cursor int) bool {
	return cursor >= loc.Span.Start.Cursor && cursor < loc.Span.End.Cursor
}

// HoverInfoQuery returns documentation/info for hover.
// Used for: textDocument/hover
var HoverInfoQuery = &Query[PositionKey, *HoverInfo]{
	Name:    "HoverInfo",
	Compute: computeHoverInfo,
}

func computeHoverInfo(db *Database, key PositionKey) (*HoverInfo, error) {
	symbol, err := Get(db, SymbolAtPositionQuery, key)
	if err != nil {
		return nil, err
	}
	if symbol == nil {
		return nil, nil
	}

	var contents strings.Builder

	switch symbol.Kind {
	case SymbolKindDefinition:
		grammar, _ := Get(db, ResolvedImportsQuery, FilePath(key.File))
		if def, ok := grammar.DefsByName[symbol.Name]; ok {
			contents.WriteString(fmt.Sprintf("**%s** (rule)\n\n", symbol.Name))
			contents.WriteString("```peg\n")
			contents.WriteString(def.String())
			contents.WriteString("\n```")

			// Check if recursive
			recursiveSet, _ := Get(db, RecursiveSetQuery, FilePath(key.File))
			if _, isRecursive := recursiveSet[symbol.Name]; isRecursive {
				contents.WriteString("\n\n*This rule is recursive.*")
			}

			// Check if it's used as a recovery rule
			recoveryRules, _ := Get(db, RecoveryRulesQuery, FilePath(key.File))
			if info, ok := recoveryRules[symbol.Name]; ok && len(info.UsageLocs) > 0 {
				contents.WriteString(fmt.Sprintf("\n\n*Recovery rule for %d error label(s).*", len(info.UsageLocs)))
			}
		}

	case SymbolKindIdentifier:
		contents.WriteString(fmt.Sprintf("**%s** (reference)\n\n", symbol.Name))
		if symbol.DefinitionLoc != nil {
			grammar, _ := Get(db, ResolvedImportsQuery, FilePath(key.File))
			if def, ok := grammar.DefsByName[symbol.Name]; ok {
				contents.WriteString("```peg\n")
				contents.WriteString(def.String())
				contents.WriteString("\n```")
			}
		} else {
			contents.WriteString("*Undefined rule*")
		}

	case SymbolKindLabel:
		contents.WriteString(fmt.Sprintf("**^%s** (error label)\n\n", symbol.Name))
		if symbol.DefinitionLoc != nil {
			contents.WriteString(fmt.Sprintf("Recovery rule: `%s`", symbol.Name))
		} else {
			contents.WriteString("*No recovery rule defined.* The parser will fail if this error occurs.")
		}

	case SymbolKindLiteral:
		contents.WriteString(fmt.Sprintf("**Literal**: `\"%s\"`", symbol.Name))

	case SymbolKindClass:
		contents.WriteString(fmt.Sprintf("**Character class**: `%s`", symbol.Name))
	}

	return &HoverInfo{
		Contents: contents.String(),
		Range:    symbol.Location,
	}, nil
}

// CompletionItemsQuery returns completion suggestions at a position.
// Used for: textDocument/completion
var CompletionItemsQuery = &Query[PositionKey, []CompletionItem]{
	Name:    "CompletionItems",
	Compute: computeCompletionItems,
}

func computeCompletionItems(db *Database, key PositionKey) ([]CompletionItem, error) {
	grammar, err := Get(db, ResolvedImportsQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}

	recursiveSet, _ := Get(db, RecursiveSetQuery, FilePath(key.File))
	recoveryRules, _ := Get(db, RecoveryRulesQuery, FilePath(key.File))

	var items []CompletionItem

	// Add all rule names
	for _, def := range grammar.Definitions {
		detail := summarizeExpression(def.Expr)
		if _, isRecursive := recursiveSet[def.Name]; isRecursive {
			detail = "(recursive) " + detail
		}

		doc := ""
		if info, ok := recoveryRules[def.Name]; ok && len(info.UsageLocs) > 0 {
			doc = fmt.Sprintf("Recovery rule for %d error label(s)", len(info.UsageLocs))
		}

		items = append(items, CompletionItem{
			Label:         def.Name,
			Kind:          CompletionKindRule,
			Detail:        detail,
			Documentation: doc,
			InsertText:    def.Name,
		})
	}

	// Add keywords
	keywords := []string{"@import", "from"}
	for _, kw := range keywords {
		items = append(items, CompletionItem{
			Label:      kw,
			Kind:       CompletionKindKeyword,
			InsertText: kw,
		})
	}

	// Add label completions (for ^)
	seenLabels := make(map[string]bool)
	labels, _ := Get(db, LabelLocationsQuery, FilePath(key.File))
	for _, label := range labels {
		if !seenLabels[label.Label] {
			seenLabels[label.Label] = true

			detail := "error label"
			if info, ok := recoveryRules[label.Label]; ok && info.HasRecovery {
				detail = "error label (has recovery)"
			}

			items = append(items, CompletionItem{
				Label:      label.Label,
				Kind:       CompletionKindLabel,
				Detail:     detail,
				InsertText: label.Label,
			})
		}
	}

	// Snippets
	snippets := []CompletionItem{
		{
			Label:      "rule",
			Kind:       CompletionKindSnippet,
			Detail:     "New rule definition",
			InsertText: "${1:RuleName} <- ${2:expression}",
		},
		{
			Label:      "import",
			Kind:       CompletionKindSnippet,
			Detail:     "Import statement",
			InsertText: "@import ${1:Name} from \"${2:./path.peg}\"",
		},
		{
			Label:      "choice",
			Kind:       CompletionKindSnippet,
			Detail:     "Choice expression",
			InsertText: "${1:a} / ${2:b}",
		},
		{
			Label:      "labeled",
			Kind:       CompletionKindSnippet,
			Detail:     "Labeled failure",
			InsertText: "${1:expr}^${2:ErrorLabel}",
		},
	}
	items = append(items, snippets...)

	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })

	return items, nil
}

// References Query

// ReferencesQuery finds all references to a symbol.  Used for:
// textDocument/references
var ReferencesQuery = &Query[ReferencesKey, []SourceLocation]{
	Name:    "References",
	Compute: computeReferences,
}

func computeReferences(db *Database, key ReferencesKey) ([]SourceLocation, error) {
	var locations []SourceLocation

	defLocs, err := Get(db, DefinitionLocationsQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}
	if defLoc, ok := defLocs[key.SymbolName]; ok {
		locations = append(locations, defLoc)
	}

	idLocs, err := Get(db, IdentifierLocationsQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}
	for _, idLoc := range idLocs {
		if idLoc.Name == key.SymbolName && !idLoc.IsDefinition {
			locations = append(locations, idLoc.Location)
		}
	}

	recoveryRules, err := Get(db, RecoveryRulesQuery, FilePath(key.File))
	if err != nil {
		return nil, err
	}
	if info, ok := recoveryRules[key.SymbolName]; ok {
		locations = append(locations, info.UsageLocs...)
	}

	return locations, nil
}

// Semantic Tokens Query

// SemanticTokensQuery returns tokens for semantic highlighting.  Used
// for: textDocument/semanticTokens
var SemanticTokensQuery = &Query[FilePath, []SemanticToken]{
	Name:    "SemanticTokens",
	Compute: computeSemanticTokens,
}

func computeSemanticTokens(db *Database, key FilePath) ([]SemanticToken, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	defLocs, err := Get(db, DefinitionLocationsQuery, key)
	if err != nil {
		return nil, err
	}
	recursiveSet, err := Get(db, RecursiveSetQuery, key)
	if err != nil {
		return nil, err
	}
	unusedRules, err := Get(db, UnusedRulesQuery, key)
	if err != nil {
		return nil, err
	}
	unusedSet := make(map[string]bool)
	for _, name := range unusedRules {
		unusedSet[name] = true
	}
	recoveryRules, err := Get(db, RecoveryRulesQuery, key)
	if err != nil {
		return nil, err
	}
	undefinedRefs, err := Get(db, UndefinedReferencesQuery, key)
	if err != nil {
		return nil, err
	}
	undefinedSet := make(map[string]bool)
	for _, ref := range undefinedRefs {
		undefinedSet[ref.Name] = true
	}

	var tokens []SemanticToken

	for _, def := range grammar.Definitions {
		// Definition name token
		defLoc := def.SourceLocation()
		nameSpan := Span{
			Start: defLoc.Span.Start,
			End: Location{
				Line:   defLoc.Span.Start.Line,
				Column: defLoc.Span.Start.Column + len(def.Name),
				Cursor: defLoc.Span.Start.Cursor + len(def.Name),
			},
		}
		nameLoc := SourceLocation{FileID: defLoc.FileID, Span: nameSpan}

		modifiers := []string{"definition"}
		if _, isRecursive := recursiveSet[def.Name]; isRecursive {
			modifiers = append(modifiers, "recursive")
		}
		if unusedSet[def.Name] {
			modifiers = append(modifiers, "unused")
		}
		if info, ok := recoveryRules[def.Name]; ok && len(info.UsageLocs) > 0 {
			modifiers = append(modifiers, "recovery")
		}

		tokens = append(tokens, SemanticToken{
			Location:  nameLoc,
			TokenType: SemanticTokenTypeDefinition,
			Modifiers: modifiers,
		})

		// Tokens in the expression
		collectExpressionTokens(def.Expr, defLocs, recursiveSet, undefinedSet, recoveryRules, &tokens)
	}

	// Sort tokens by position
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].Location.Span.Start.Cursor < tokens[j].Location.Span.Start.Cursor
	})

	return tokens, nil
}

// collectExpressionTokens recursively collects semantic tokens from
// an expression.
func collectExpressionTokens(
	expr AstNode,
	defLocs map[string]SourceLocation,
	recursiveSet map[string]struct{},
	undefinedSet map[string]bool,
	recoveryRules map[string]*RecoveryRuleInfo,
	tokens *[]SemanticToken,
) {
	if expr == nil {
		return
	}

	Inspect(expr, func(n AstNode) bool {
		if n == nil {
			return true
		}

		loc := n.SourceLocation()

		switch node := n.(type) {
		case *IdentifierNode:
			modifiers := []string{}
			if _, ok := defLocs[node.Value]; !ok {
				modifiers = append(modifiers, "unresolved")
			} else {
				if _, isRecursive := recursiveSet[node.Value]; isRecursive {
					modifiers = append(modifiers, "recursive")
				}
				if info, ok := recoveryRules[node.Value]; ok && len(info.UsageLocs) > 0 {
					modifiers = append(modifiers, "recovery")
				}
			}

			*tokens = append(*tokens, SemanticToken{
				Location:  loc,
				TokenType: SemanticTokenTypeIdentifier,
				Modifiers: modifiers,
			})
			return false // Don't recurse further

		case *LabeledNode:
			// Token for the label itself The label is
			// after ^, so we need to compute its location
			exprEnd := node.Expr.SourceLocation().Span.End

			labelStart := Location{
				Line:   exprEnd.Line,
				Column: exprEnd.Column + 1, // +1 for ^
				Cursor: exprEnd.Cursor + 1,
			}
			labelEnd := Location{
				Line:   labelStart.Line,
				Column: labelStart.Column + len(node.Label),
				Cursor: labelStart.Cursor + len(node.Label),
			}
			labelLoc := SourceLocation{
				FileID: loc.FileID,
				Span:   Span{Start: labelStart, End: labelEnd},
			}

			modifiers := []string{}
			if info, ok := recoveryRules[node.Label]; ok {
				if !info.HasRecovery {
					modifiers = append(modifiers, "unresolved")
				}
			}

			*tokens = append(*tokens, SemanticToken{
				Location:  labelLoc,
				TokenType: SemanticTokenTypeLabel,
				Modifiers: modifiers,
			})

			// Recurse into the labeled expression
			collectExpressionTokens(node.Expr, defLocs, recursiveSet, undefinedSet, recoveryRules, tokens)
			return false

		case *LiteralNode:
			*tokens = append(*tokens, SemanticToken{
				Location:  loc,
				TokenType: SemanticTokenTypeLiteral,
				Modifiers: nil,
			})
			return false

		case *ClassNode, *CharsetNode, *RangeNode:
			*tokens = append(*tokens, SemanticToken{
				Location:  loc,
				TokenType: SemanticTokenTypeClass,
				Modifiers: nil,
			})
			return false
		}

		return true
	})
}
