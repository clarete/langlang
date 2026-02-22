package langlang

import "fmt"

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
// Expected tree structure:
//   TypeDecl(Sequence[Identifier("name"), CtorList(Sequence[CtorDecl, ...])])
func ParseTypeDecl(tree Tree, id NodeID) (*TypeDecl, error) {
	children := namedNodeChildren(tree, id)
	if len(children) < 2 {
		return nil, fmt.Errorf("TypeDecl: expected at least 2 children (name + ctors), got %d", len(children))
	}

	name := tree.Text(children[0])
	ctorListID := children[1]

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
	if len(children) < 1 {
		return nil, fmt.Errorf("CtorDecl: expected at least 1 child (name)")
	}

	name := tree.Text(children[0])
	var fields []*Field

	for _, childID := range children[1:] {
		if tree.Type(childID) == NodeType_Node && tree.Name(childID) == "FieldList" {
			var err error
			fields, err = parseFieldList(tree, childID)
			if err != nil {
				return nil, err
			}
		}
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
	if len(children) < 2 {
		return nil, fmt.Errorf("FieldDecl: expected 2 children (name, type)")
	}
	return &Field{
		Name:     tree.Text(children[0]),
		TypeName: tree.Text(children[1]),
	}, nil
}

// ParseRewriteDef converts a parse tree node for a rewrite definition
// into a RewriteRuleSet.
func ParseRewriteDef(tree Tree, id NodeID) (*RewriteRuleSet, string, error) {
	children := namedNodeChildren(tree, id)
	if len(children) < 2 {
		return nil, "", fmt.Errorf("RewriteDef: expected at least 2 children")
	}

	idx := 0
	exhaustiveType := ""

	// Check for @exhaustive annotation
	if tree.Type(children[0]) == NodeType_Node && tree.Name(children[0]) == "Exhaustive" {
		exChildren := tree.Children(children[0])
		if len(exChildren) > 0 {
			exhaustiveType = tree.Text(exChildren[0])
		}
		idx++
	}

	name := tree.Text(children[idx])
	idx++

	altListID := children[idx]
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
	if len(children) < 2 {
		return nil, fmt.Errorf("RewriteAlt: expected 2 children (pattern, construction)")
	}

	pat, err := parseRwPattern(tree, children[0])
	if err != nil {
		return nil, err
	}

	con, err := parseRwConstruction(tree, children[1])
	if err != nil {
		return nil, err
	}

	return &RewriteRule{
		Name:    ruleName,
		Pattern: pat,
		Constr:  con,
	}, nil
}

func parseRwPattern(tree Tree, id NodeID) (RewritePattern, error) {
	if tree.Type(id) != NodeType_Node {
		return nil, fmt.Errorf("expected pattern node, got %s", tree.Type(id))
	}

	switch tree.Name(id) {
	case "RwPatNamed":
		return parseRwPatNamed(tree, id)
	case "RwPatVar":
		return PatVar{Name: tree.Text(id)}, nil
	case "RwPatWild":
		return PatWild{}, nil
	case "RwPatStr":
		return PatStr{Text: tree.Text(id)}, nil
	case "RwPatSeq":
		return parseRwPatSeq(tree, id)
	case "RwPattern":
		// Wrapper node, descend
		child, _ := tree.Child(id)
		return parseRwPattern(tree, child)
	default:
		return nil, fmt.Errorf("unknown pattern node: %s", tree.Name(id))
	}
}

func parseRwPatNamed(tree Tree, id NodeID) (RewritePattern, error) {
	children := namedNodeChildren(tree, id)
	if len(children) < 2 {
		return nil, fmt.Errorf("RwPatNamed: expected at least 2 children")
	}

	name := tree.Text(children[0])
	body, err := parseRwPattern(tree, children[1])
	if err != nil {
		return nil, err
	}

	return PatNamed{NodeName: name, Body: body}, nil
}

func parseRwPatSeq(tree Tree, id NodeID) (RewritePattern, error) {
	var elems []RewritePattern
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node {
			elem, err := parseRwPattern(tree, childID)
			if err != nil {
				return nil, err
			}
			elems = append(elems, elem)
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
		return ConVar{Name: tree.Text(id)}, nil
	case "RwConStr":
		return ConStr{Text: tree.Text(id)}, nil
	case "RwConSeq":
		return parseRwConSeq(tree, id)
	case "RwConCall":
		return parseRwConCall(tree, id)
	case "RwConstruction":
		child, _ := tree.Child(id)
		return parseRwConstruction(tree, child)
	default:
		return nil, fmt.Errorf("unknown construction node: %s", tree.Name(id))
	}
}

func parseRwConNamed(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)
	if len(children) < 2 {
		return nil, fmt.Errorf("RwConNamed: expected at least 2 children")
	}

	name := tree.Text(children[0])
	body, err := parseRwConstruction(tree, children[1])
	if err != nil {
		return nil, err
	}

	return ConNamed{NodeName: name, Body: body}, nil
}

func parseRwConSeq(tree Tree, id NodeID) (RewriteConstruction, error) {
	var elems []RewriteConstruction
	for _, childID := range namedNodeChildren(tree, id) {
		if tree.Type(childID) == NodeType_Node {
			elem, err := parseRwConstruction(tree, childID)
			if err != nil {
				return nil, err
			}
			elems = append(elems, elem)
		}
	}
	return ConSeq{Elems: elems}, nil
}

func parseRwConCall(tree Tree, id NodeID) (RewriteConstruction, error) {
	children := namedNodeChildren(tree, id)
	if len(children) < 2 {
		return nil, fmt.Errorf("RwConCall: expected at least 2 children")
	}

	// A construction call like expr(?e) is syntactic sugar for applying
	// a rewrite rule to a variable. At the codegen level this becomes
	// a function call: rw.expr(tree, ?e).
	name := tree.Text(children[0])
	arg, err := parseRwConstruction(tree, children[1])
	if err != nil {
		return nil, err
	}

	return ConNamed{NodeName: name, Body: arg}, nil
}
