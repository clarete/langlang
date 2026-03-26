package extract

import (
	"fmt"
	"strings"

	langlang "github.com/clarete/langlang/go"
)

// AnalyzeGrammar loads a PEG grammar and returns a map of rule names to
// RuleInfo, describing each rule's structure for extraction codegen.
func AnalyzeGrammar(grammarPath string) (map[string]RuleInfo, error) {
	cfg := langlang.NewConfig()
	loader := langlang.NewRelativeImportLoader()
	db := langlang.NewDatabase(cfg, loader)

	grammar, err := langlang.QueryAST(db, grammarPath)
	if err != nil {
		return nil, fmt.Errorf("query AST: %w", err)
	}

	bytecode, err := langlang.QueryBytecode(db, grammarPath)
	if err != nil {
		return nil, fmt.Errorf("query bytecode: %w", err)
	}

	rules := make(map[string]RuleInfo, len(grammar.Definitions))
	for _, def := range grammar.Definitions {
		ri := classifyRule(def)
		if id, ok := bytecode.StringID(def.Name); ok {
			ri.NameID = int32(id)
		} else {
			ri.NameID = -1
		}
		rules[def.Name] = ri
	}

	return rules, nil
}

func classifyRule(def *langlang.DefinitionNode) RuleInfo {
	ri := RuleInfo{Name: def.Name}
	expr := unwrapTransparent(def.Expr)

	switch e := expr.(type) {
	case *langlang.SequenceNode:
		ri.Kind = RuleSequence
		ri.Children = classifySequenceChildren(e)

	case *langlang.ChoiceNode:
		ri.Kind = RuleChoice
		ri.Choices = flattenChoices(e)

	case *langlang.ZeroOrMoreNode:
		ri.Kind = RuleRepeat
		if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
			ri.Inner = id.Value
		}

	case *langlang.OneOrMoreNode:
		ri.Kind = RuleRepeat
		if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
			ri.Inner = id.Value
		}

	case *langlang.OptionalNode:
		ri.Kind = RuleOptional
		if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
			ri.Inner = id.Value
		}

	case *langlang.IdentifierNode:
		ri.Kind = RuleAlias
		ri.Inner = e.Value

	default:
		ri.Kind = RuleLeaf
	}

	return ri
}

// unwrapTransparent strips AST wrappers that don't affect tree structure.
func unwrapTransparent(n langlang.AstNode) langlang.AstNode {
	for {
		switch e := n.(type) {
		case *langlang.LexNode:
			n = e.Expr
		case *langlang.LabeledNode:
			n = e.Expr
		case *langlang.CaptureNode:
			n = e.Expr
		default:
			return n
		}
	}
}

func classifySequenceChildren(seq *langlang.SequenceNode) []RuleChild {
	var children []RuleChild
	for i, item := range seq.Items {
		child := RuleChild{Index: i}
		inner := unwrapTransparent(item)

		switch e := inner.(type) {
		case *langlang.IdentifierNode:
			child.RuleName = e.Value
		case *langlang.LiteralNode:
			child.IsLiteral = true
		case *langlang.OptionalNode:
			if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
				child.RuleName = id.Value
			} else if seq2, ok := unwrapTransparent(e.Expr).(*langlang.SequenceNode); ok {
				nested := classifySequenceChildren(seq2)
				children = append(children, nested...)
				continue
			} else {
				child.IsLiteral = true
			}
		case *langlang.ZeroOrMoreNode:
			if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
				child.RuleName = id.Value
				child.Repeated = true
			} else if seq2, ok := unwrapTransparent(e.Expr).(*langlang.SequenceNode); ok {
				nested := classifySequenceChildren(seq2)
				for j := range nested {
					nested[j].Repeated = true
				}
				children = append(children, nested...)
				continue
			}
		case *langlang.OneOrMoreNode:
			if id, ok := unwrapTransparent(e.Expr).(*langlang.IdentifierNode); ok {
				child.RuleName = id.Value
				child.Repeated = true
			} else if seq2, ok := unwrapTransparent(e.Expr).(*langlang.SequenceNode); ok {
				nested := classifySequenceChildren(seq2)
				for j := range nested {
					nested[j].Repeated = true
				}
				children = append(children, nested...)
				continue
			}
		case *langlang.SequenceNode:
			nested := classifySequenceChildren(e)
			children = append(children, nested...)
			continue
		default:
			child.IsLiteral = true
		}

		children = append(children, child)
	}
	return children
}

// flattenChoices collects all alternatives from a binary right-associative
// ChoiceNode tree into a flat list of names. Non-identifier alternatives
// (like literals 'true', 'false', 'null') are included as empty strings.
func flattenChoices(c *langlang.ChoiceNode) []string {
	var choices []string
	flattenChoicesInto(c, &choices)
	return choices
}

func flattenChoicesInto(node langlang.AstNode, out *[]string) {
	inner := unwrapTransparent(node)
	switch e := inner.(type) {
	case *langlang.ChoiceNode:
		flattenChoicesInto(e.Left, out)
		flattenChoicesInto(e.Right, out)
	case *langlang.IdentifierNode:
		*out = append(*out, e.Value)
	case *langlang.LiteralNode:
		*out = append(*out, ChoiceLiteralPrefix+e.Value)
	default:
		*out = append(*out, "")
	}
}

// ChoiceLiteralPrefix marks a choice alternative as a literal string match
// rather than a rule reference. Used in RuleInfo.Choices.
const ChoiceLiteralPrefix = "lit:"

// IsChoiceLiteral reports whether a choice entry is a literal and returns its text.
func IsChoiceLiteral(choice string) (string, bool) {
	if strings.HasPrefix(choice, ChoiceLiteralPrefix) {
		return choice[len(ChoiceLiteralPrefix):], true
	}
	return "", false
}
