package langlang

import (
	"fmt"
	"strings"
)

func stripQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// trimIdentifier extracts a clean identifier from text that may have
// a prefix (like "?" for variables) consumed by a lexified sub-expression.
func trimIdentifier(s string) string {
	if idx := strings.IndexFunc(s, func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
	}); idx > 0 {
		return s[idx:]
	}
	return s
}

// namedNodeChildren returns the effective children of a named node.
// In langlang's tree structure, a NodeType_Node has a single child
// which is typically a Sequence or another node. This helper
// extracts the meaningful children:
//   - If the child is a Sequence, returns the sequence's children.
//   - If the child is a Node, returns it as a single-element slice.
//   - Otherwise returns the child as-is.
func namedNodeChildren(tree Tree, id NodeID) []NodeID {
	child, ok := tree.Child(id)
	if !ok {
		return nil
	}
	if tree.Type(child) == NodeType_Sequence {
		return tree.Children(child)
	}
	return []NodeID{child}
}

// ParseTypeDecl converts a parse tree node for @type into a TypeDecl.
func ParseTypeDecl(tree Tree, id NodeID) (*TypeDecl, error) {
	children := namedNodeChildren(tree, id)

	var name string
	var ctorListID NodeID
	foundCtor := false

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Identifier":
			if name == "" {
				name = tree.Text(childID)
			}
		case "CtorList":
			ctorListID = childID
			foundCtor = true
		}
	}

	if name == "" || !foundCtor {
		return nil, fmt.Errorf("TypeDecl: missing name or constructor list")
	}

	ctors, err := parseCtorList(tree, ctorListID)
	if err != nil {
		return nil, err
	}

	return &TypeDecl{
		Name:         name,
		Constructors: ctors,
	}, nil
}

func parseCtorList(tree Tree, id NodeID) ([]*Constructor, error) {
	var ctors []*Constructor
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "CtorDecl" {
			ctor, err := parseCtorDecl(tree, childID)
			if err != nil {
				return nil, err
			}
			ctors = append(ctors, ctor)
		}
	}
	return ctors, nil
}

func parseCtorDecl(tree Tree, id NodeID) (*Constructor, error) {
	children := namedNodeChildren(tree, id)

	var name string
	var fields []*Field

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Identifier":
			if name == "" {
				name = tree.Text(childID)
			}
		case "FieldList":
			var err error
			fields, err = parseFieldList(tree, childID)
			if err != nil {
				return nil, err
			}
		}
	}

	if name == "" {
		return nil, fmt.Errorf("CtorDecl: missing name")
	}

	return &Constructor{
		Name:   name,
		Fields: fields,
	}, nil
}

func parseFieldList(tree Tree, id NodeID) ([]*Field, error) {
	var fields []*Field
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "FieldDecl" {
			field, err := parseFieldDecl(tree, childID)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		}
	}
	return fields, nil
}

func parseFieldDecl(tree Tree, id NodeID) (*Field, error) {
	children := namedNodeChildren(tree, id)

	var name string
	var typeName string

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Identifier":
			if name == "" {
				name = tree.Text(childID)
			}
		case "TypeRef":
			typeName = tree.Text(childID)
		}
	}

	if name == "" {
		return nil, fmt.Errorf("FieldDecl: missing name")
	}

	return &Field{
		Name:     name,
		TypeName: typeName,
	}, nil
}

// ParseRewriteDef converts a parse tree node for a rewrite definition
// into a RewriteRuleSet.
func ParseRewriteDef(tree Tree, id NodeID) (*RewriteRuleSet, string, error) {
	children := namedNodeChildren(tree, id)

	var name string
	exhaustiveType := ""
	var altListID NodeID
	foundAltList := false

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Exhaustive":
			for _, exChild := range namedNodeChildren(tree, childID) {
				if tree.Type(exChild) == NodeType_Node && tree.Name(exChild) == "Identifier" {
					exhaustiveType = tree.Text(exChild)
				}
			}
		case "Identifier":
			if name == "" {
				name = tree.Text(childID)
			}
		case "RewriteAltList":
			altListID = childID
			foundAltList = true
		}
	}

	if name == "" || !foundAltList {
		return nil, "", fmt.Errorf("RewriteDef: missing name or alt list")
	}

	rules, err := parseRewriteAltList(tree, altListID, name)
	if err != nil {
		return nil, "", err
	}

	return &RewriteRuleSet{
		Name:  name,
		Rules: rules,
	}, exhaustiveType, nil
}

func parseRewriteAltList(tree Tree, id NodeID, ruleName string) ([]*RewriteRule, error) {
	var rules []*RewriteRule
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "RewriteAlt" {
			rule, err := parseRewriteAlt(tree, childID, ruleName)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func parseRewriteAlt(tree Tree, id NodeID, ruleName string) (*RewriteRule, error) {
	children := namedNodeChildren(tree, id)

	var patNode, conNode NodeID
	foundPat, foundCon := false, false

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		name := tree.Name(childID)
		if !foundPat && isPatternNodeName(name) {
			patNode = childID
			foundPat = true
		} else if foundPat && !foundCon && isConstructionNodeName(name) {
			conNode = childID
			foundCon = true
		}
	}

	if !foundPat || !foundCon {
		return nil, fmt.Errorf("RewriteAlt: missing pattern or construction")
	}

	pat, err := parseRwPattern(tree, patNode)
	if err != nil {
		return nil, err
	}

	con, err := parseRwConstruction(tree, conNode)
	if err != nil {
		return nil, err
	}

	return &RewriteRule{
		Name:    ruleName,
		Pattern: pat,
		Constr:  con,
	}, nil
}

func isPatternNodeName(name string) bool {
	switch name {
	case "RwPatNamed", "RwPatVar", "RwPatWild", "RwPatStr", "RwPatSeq", "RwPattern":
		return true
	}
	return false
}

func isConstructionNodeName(name string) bool {
	switch name {
	case "RwConNamed", "RwConVar", "RwConStr", "RwConSeq", "RwConCall",
		"RwConEach", "RwConLen", "RwConFoldl", "RwConstruction":
		return true
	}
	return false
}

func parseRwPattern(tree Tree, id NodeID) (RewritePattern, error) {
	if tree.Type(id) != NodeType_Node {
		return nil, fmt.Errorf("expected pattern node, got %s", tree.Type(id))
	}

	switch tree.Name(id) {
	case "RwPatNamed":
		return parseRwPatNamed(tree, id)
	case "RwPatVar":
		return PatVar{Name: trimIdentifier(tree.Text(id))}, nil
	case "RwPatWild":
		return PatWild{}, nil
	case "RwPatStr":
		return PatStr{Text: stripQuotes(tree.Text(id))}, nil
	case "RwPatSeq":
		return parseRwPatSeq(tree, id)
	case "RwPattern":
		child, _ := tree.Child(id)
		return parseRwPattern(tree, child)
	case "Spacing":
		return nil, nil // skip spacing nodes
	default:
		return nil, fmt.Errorf("unknown pattern node: %s", tree.Name(id))
	}
}

func parseRwPatNamed(tree Tree, id NodeID) (RewritePattern, error) {
	children := namedNodeChildren(tree, id)

	var name string
	for _, childID := range children {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "Identifier" {
			name = tree.Text(childID)
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("RwPatNamed: missing name")
	}

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "RwPatFields":
			return parseRwPatFieldsAsNamed(tree, name, childID)
		case "RwPatNamed", "RwPatVar", "RwPatWild", "RwPatStr", "RwPatSeq", "RwPattern":
			body, err := parseRwPattern(tree, childID)
			if err != nil {
				return nil, err
			}
			return PatNamed{NodeName: name, Body: body}, nil
		}
	}

	return nil, fmt.Errorf("RwPatNamed: missing body for %s", name)
}

// parseRwPatFieldsAsNamed converts field patterns like Binary(Op: ?op, Left: ?l)
// into PatNamed{NodeName: "Binary", Body: PatSeq{[?op, ?l]}}.
// Field names are used for documentation; the tree stores children positionally.
func parseRwPatFieldsAsNamed(tree Tree, name string, fieldsNode NodeID) (RewritePattern, error) {
	var elems []RewritePattern
	for _, childID := range namedNodeChildren(tree, fieldsNode) {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "RwFieldPat" {
			for _, fc := range namedNodeChildren(tree, childID) {
				if tree.Type(fc) == NodeType_Node && isPatternNodeName(tree.Name(fc)) {
					pat, err := parseRwPattern(tree, fc)
					if err != nil {
						return nil, err
					}
					elems = append(elems, pat)
					break
				}
			}
		}
	}
	if len(elems) == 1 {
		return PatNamed{NodeName: name, Body: elems[0]}, nil
	}
	return PatNamed{NodeName: name, Body: PatSeq{Elems: elems}}, nil
}

func parseRwPatSeq(tree Tree, id NodeID) (RewritePattern, error) {
	var elems []RewritePattern
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node && isPatternNodeName(tree.Name(childID)) {
			elem, err := parseRwPattern(tree, childID)
			if err != nil {
				return nil, err
			}
			if elem != nil {
				elems = append(elems, elem)
			}
		}
	}
	return PatSeq{Elems: elems}, nil
}

func parseRwConstruction(tree Tree, id NodeID) (RewriteConstruction, error) {
	if tree.Type(id) != NodeType_Node {
		return nil, fmt.Errorf("expected construction node, got %s", tree.Type(id))
	}

	switch tree.Name(id) {
	case "RwConNamed":
		return parseRwConNamed(tree, id)
	case "RwConVar":
		return ConVar{Name: trimIdentifier(tree.Text(id))}, nil
	case "RwConStr":
		return ConStr{Text: stripQuotes(tree.Text(id))}, nil
	case "RwConSeq":
		return parseRwConSeq(tree, id)
	case "RwConCall":
		return parseRwConCall(tree, id)
	case "RwConEach":
		return parseRwConEach(tree, id)
	case "RwConLen":
		return parseRwConLen(tree, id)
	case "RwConFoldl":
		return parseRwConFoldl(tree, id)
	case "RwConstruction":
		child, _ := tree.Child(id)
		return parseRwConstruction(tree, child)
	case "Spacing":
		return nil, nil // skip spacing nodes
	default:
		return nil, fmt.Errorf("unknown construction node: %s", tree.Name(id))
	}
}

func parseRwConNamed(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)

	var name string
	for _, childID := range children {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "Identifier" {
			name = tree.Text(childID)
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("RwConNamed: missing name")
	}

	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "RwConFields":
			return parseRwConFieldsAsNamed(tree, name, childID)
		case "RwConNamed", "RwConVar", "RwConStr", "RwConSeq", "RwConCall", "RwConstruction":
			body, err := parseRwConstruction(tree, childID)
			if err != nil {
				return nil, err
			}
			return ConNamed{NodeName: name, Body: body}, nil
		}
	}

	return nil, fmt.Errorf("RwConNamed: missing body for %s", name)
}

func parseRwConFieldsAsNamed(tree Tree, name string, fieldsNode NodeID) (RewriteConstruction, error) {
	var elems []RewriteConstruction
	for _, childID := range namedNodeChildren(tree, fieldsNode) {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "RwConField" {
			for _, fc := range namedNodeChildren(tree, childID) {
				if tree.Type(fc) == NodeType_Node && isConstructionNodeName(tree.Name(fc)) {
					con, err := parseRwConstruction(tree, fc)
					if err != nil {
						return nil, err
					}
					elems = append(elems, con)
					break
				}
			}
		}
	}
	if len(elems) == 1 {
		return ConNamed{NodeName: name, Body: elems[0]}, nil
	}
	return ConNamed{NodeName: name, Body: ConSeq{Elems: elems}}, nil
}

func parseRwConSeq(tree Tree, id NodeID) (RewriteConstruction, error) {
	var elems []RewriteConstruction
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node && isConstructionNodeName(tree.Name(childID)) {
			elem, err := parseRwConstruction(tree, childID)
			if err != nil {
				return nil, err
			}
			if elem != nil {
				elems = append(elems, elem)
			}
		}
	}
	return ConSeq{Elems: elems}, nil
}

func parseRwConCall(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)

	var name string
	var args []RewriteConstruction
	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Identifier":
			if name == "" {
				name = tree.Text(childID)
			}
		default:
			if isConstructionNodeName(tree.Name(childID)) {
				arg, err := parseRwConstruction(tree, childID)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
		}
	}

	if name == "" {
		return nil, fmt.Errorf("RwConCall: missing name")
	}
	return ConCall{RuleName: name, Args: args}, nil
}

// parseRwConEach parses: each(Identifier, RwConstruction)
func parseRwConEach(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)
	var ruleName string
	var seqArg RewriteConstruction
	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Identifier":
			if ruleName == "" {
				ruleName = tree.Text(childID)
			}
		default:
			if isConstructionNodeName(tree.Name(childID)) && seqArg == nil {
				var err error
				seqArg, err = parseRwConstruction(tree, childID)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if ruleName == "" || seqArg == nil {
		return nil, fmt.Errorf("RwConEach: missing rule name or sequence arg")
	}
	return ConEach{RuleName: ruleName, SeqArg: seqArg}, nil
}

// parseRwConLen parses: len(RwConstruction)
func parseRwConLen(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)
	for _, childID := range children {
		if tree.Type(childID) == NodeType_Node && isConstructionNodeName(tree.Name(childID)) {
			arg, err := parseRwConstruction(tree, childID)
			if err != nil {
				return nil, err
			}
			return ConLen{SeqArg: arg}, nil
		}
	}
	return nil, fmt.Errorf("RwConLen: missing argument")
}

// parseRwConFoldl parses: foldl(Identifier, Identifier, RwConstruction)
func parseRwConFoldl(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)
	var ctorName, ruleName string
	var seqArg RewriteConstruction
	for _, childID := range children {
		if tree.Type(childID) != NodeType_Node {
			continue
		}
		switch tree.Name(childID) {
		case "Identifier":
			if ctorName == "" {
				ctorName = tree.Text(childID)
			} else if ruleName == "" {
				ruleName = tree.Text(childID)
			}
		default:
			if isConstructionNodeName(tree.Name(childID)) && seqArg == nil {
				var err error
				seqArg, err = parseRwConstruction(tree, childID)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if ctorName == "" || ruleName == "" || seqArg == nil {
		return nil, fmt.Errorf("RwConFoldl: missing ctor name, rule name, or sequence arg")
	}
	return ConFoldl{CtorName: ctorName, RuleName: ruleName, SeqArg: seqArg}, nil
}

// RewriteFile holds the parsed contents of a .peg file with @type and <~ declarations.
type RewriteFile struct {
	Types         []*TypeDecl
	RuleSets      []*RewriteRuleSet
	ExhaustiveMap map[string]string // rule name -> type name
}

// ParseRewriteFile walks a parse tree produced by the rewrite_syntax.peg grammar
// and extracts all @type declarations and <~ rewrite definitions.
func ParseRewriteFile(tree Tree, root NodeID) (*RewriteFile, error) {
	result := &RewriteFile{
		ExhaustiveMap: make(map[string]string),
	}

	for _, childID := range allNamedChildren(tree, root) {
		switch tree.Name(childID) {
		case "TypeDecl":
			td, err := ParseTypeDecl(tree, childID)
			if err != nil {
				return nil, fmt.Errorf("parsing @type: %w", err)
			}
			result.Types = append(result.Types, td)

		case "RewriteDef":
			rs, exhaustiveType, err := ParseRewriteDef(tree, childID)
			if err != nil {
				return nil, fmt.Errorf("parsing rewrite def %q: %w", tree.Text(childID), err)
			}
			result.RuleSets = append(result.RuleSets, rs)
			if exhaustiveType != "" {
				result.ExhaustiveMap[rs.Name] = exhaustiveType
			}
		}
	}

	return result, nil
}

// allNamedChildren collects all NodeType_Node children at any nesting
// depth within sequences under the given root.
func allNamedChildren(tree Tree, id NodeID) []NodeID {
	var result []NodeID
	switch tree.Type(id) {
	case NodeType_Node:
		child, ok := tree.Child(id)
		if ok {
			result = append(result, allNamedChildren(tree, child)...)
		}
	case NodeType_Sequence:
		for _, childID := range tree.Children(id) {
			if tree.Type(childID) == NodeType_Node {
				result = append(result, childID)
			} else if tree.Type(childID) == NodeType_Sequence {
				result = append(result, allNamedChildren(tree, childID)...)
			}
		}
	}
	return result
}
