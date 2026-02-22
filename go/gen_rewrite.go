package langlang

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
)

// GenGoRewriteOptions configures the Go code generator for rewrite types.
type GenGoRewriteOptions struct {
	PackageName string
}

// GenGoTypes generates Go type definitions from @type declarations.
// Each TypeDecl becomes a Go interface (sum type) and each Constructor
// becomes a Go struct implementing that interface.
func GenGoTypes(types []*TypeDecl, opt GenGoRewriteOptions) (string, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "package %s\n\n", opt.PackageName)

	for _, typ := range types {
		markerMethod := unexport(typ.Name)

		// Interface (sum type)
		fmt.Fprintf(&buf, "type %s interface {\n", typ.Name)
		fmt.Fprintf(&buf, "\t%s()\n", markerMethod)
		fmt.Fprintf(&buf, "}\n\n")

		// Constructors (product types)
		for _, ctor := range typ.Constructors {
			fmt.Fprintf(&buf, "type %s struct {\n", ctor.Name)
			for _, f := range ctor.Fields {
				fmt.Fprintf(&buf, "\t%s %s\n", f.Name, goFieldType(f.TypeName))
			}
			fmt.Fprintf(&buf, "}\n\n")
			fmt.Fprintf(&buf, "func (*%s) %s() {}\n\n", ctor.Name, markerMethod)
		}
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("gofmt: %w", err)
	}
	return string(src), nil
}

func goFieldType(typeName string) string {
	if typeName == "string" {
		return "string"
	}
	// list[T] -> []T
	if strings.HasPrefix(typeName, "list[") && strings.HasSuffix(typeName, "]") {
		inner := typeName[5 : len(typeName)-1]
		return "[]" + goFieldType(inner)
	}
	return typeName
}

func unexport(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// GenGoRewriteFunctions generates Go functions from <~ rewrite rules.
// Each RewriteRuleSet becomes a Go function with a switch/case over
// constructor names, matching the pattern from the plan's example:
//
//	func (p *Rewriter) expr(tree Tree, id NodeID) Asm {
//	    name := tree.Name(id)
//	    switch name {
//	    case "Binary": ...
//	    }
//	}
func GenGoRewriteFunctions(rules []*RewriteRuleSet, opt GenGoRewriteOptions) (string, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "package %s\n\n", opt.PackageName)

	for _, ruleSet := range rules {
		emitRewriteFunction(&buf, ruleSet)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("gofmt: %w", err)
	}
	return string(src), nil
}

func emitRewriteFunction(buf *bytes.Buffer, ruleSet *RewriteRuleSet) {
	fmt.Fprintf(buf, "func (rw *Rewriter) %s(tree Tree, id NodeID) interface{} {\n", ruleSet.Name)
	fmt.Fprintf(buf, "\tname := tree.Name(id)\n")
	fmt.Fprintf(buf, "\tswitch name {\n")

	for _, rule := range ruleSet.Rules {
		pn, ok := rule.Pattern.(PatNamed)
		if !ok {
			continue
		}
		fmt.Fprintf(buf, "\tcase %q:\n", pn.NodeName)
		emitPatternBindings(buf, pn.Body, "id", "\t\t")
		fmt.Fprintf(buf, "\t\treturn %s\n", emitConstruction(rule.Constr))
	}

	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\tpanic(\"non-exhaustive match in %s\")\n", ruleSet.Name)
	fmt.Fprintf(buf, "}\n\n")
}

func emitPatternBindings(buf *bytes.Buffer, pat RewritePattern, parentVar string, indent string) {
	switch p := pat.(type) {
	case PatVar:
		fmt.Fprintf(buf, "%s%s := tree.Text(%s)\n", indent, p.Name, parentVar)
	case PatSeq:
		fmt.Fprintf(buf, "%schildren := tree.Children(%s)\n", indent, parentVar)
		for i, elem := range p.Elems {
			childVar := fmt.Sprintf("child%d", i)
			fmt.Fprintf(buf, "%s%s := children[%d]\n", indent, childVar, i)
			emitPatternBindings(buf, elem, childVar, indent)
		}
	case PatNamed:
		childVar := fmt.Sprintf("%s_%s", parentVar, strings.ToLower(p.NodeName))
		fmt.Fprintf(buf, "%s// navigate into %s\n", indent, p.NodeName)
		fmt.Fprintf(buf, "%s%s, _ := tree.Child(%s)\n", indent, childVar, parentVar)
		emitPatternBindings(buf, p.Body, childVar, indent)
	}
}

func emitConstruction(con RewriteConstruction) string {
	switch c := con.(type) {
	case ConVar:
		return c.Name
	case ConStr:
		return fmt.Sprintf("%q", c.Text)
	case ConNamed:
		return fmt.Sprintf("&%s{%s}", c.NodeName, emitConstruction(c.Body))
	case ConSeq:
		parts := make([]string, len(c.Elems))
		for i, e := range c.Elems {
			parts[i] = emitConstruction(e)
		}
		return "[]interface{}{" + strings.Join(parts, ", ") + "}"
	default:
		return "nil"
	}
}
