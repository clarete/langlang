package extract

import (
	"fmt"
	"strings"
)

// emitExtractFunction generates a complete extraction function for one struct.
// The generated code uses arena-direct access via *tree and nameID constants.
func emitExtractFunction(si StructInfo, rules map[string]RuleInfo, exported bool) string {
	var buf strings.Builder
	prefix := "extract"
	if exported {
		prefix = "Extract"
	}
	funcName := prefix + si.Name

	fmt.Fprintf(&buf, "func %s(t *tree, id NodeID) (%s, error) {\n", funcName, si.Name)
	fmt.Fprintf(&buf, "\tvar out %s\n", si.Name)

	if isChoiceStruct(&si) {
		emitChoiceBody(&buf, si, rules)
	} else {
		emitSequenceBody(&buf, si, rules)
	}

	fmt.Fprintf(&buf, "\treturn out, nil\n")
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

func emitChoiceBody(buf *strings.Builder, si StructInfo, rules map[string]RuleInfo) {
	// For a choice struct, the node's single child determines which alternative matched
	fmt.Fprintf(buf, "\tchild, ok := t.Child(id)\n")
	fmt.Fprintf(buf, "\tif !ok {\n")
	fmt.Fprintf(buf, "\t\treturn out, fmt.Errorf(\"%s: no child\")\n", si.Name)
	fmt.Fprintf(buf, "\t}\n\n")
	fmt.Fprintf(buf, "\tswitch {\n")

	for _, f := range si.Fields {
		if f.Kind != FieldOptional {
			continue
		}
		innerType := strings.TrimPrefix(f.GoType, "*")
		if innerType == "string" {
			fmt.Fprintf(buf, "\tcase t.IsNamed(child, _nameID_%s):\n", f.LLTag)
			fmt.Fprintf(buf, "\t\ts := t.Text(child)\n")
			fmt.Fprintf(buf, "\t\tout.%s = &s\n", f.GoName)
		} else {
			fmt.Fprintf(buf, "\tcase t.IsNamed(child, _nameID_%s):\n", f.LLTag)
			fmt.Fprintf(buf, "\t\tval, err := Extract%s(t, child)\n", innerType)
			fmt.Fprintf(buf, "\t\tif err != nil {\n")
			fmt.Fprintf(buf, "\t\t\treturn out, err\n")
			fmt.Fprintf(buf, "\t\t}\n")
			fmt.Fprintf(buf, "\t\tout.%s = &val\n", f.GoName)
		}
	}

	fmt.Fprintf(buf, "\tcase t.Type(child) == NodeType_String:\n")
	fmt.Fprintf(buf, "\t\t// literal alternative (e.g., 'true', 'false', 'null')\n")
	fmt.Fprintf(buf, "\t}\n")
}

func emitSequenceBody(buf *strings.Builder, si StructInfo, rules map[string]RuleInfo) {
	// Walk children, matching named nodes by nameID
	fmt.Fprintf(buf, "\tt.Visit(id, func(cid NodeID) bool {\n")
	fmt.Fprintf(buf, "\t\tif cid == id {\n")
	fmt.Fprintf(buf, "\t\t\treturn true\n")
	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\tif t.Type(cid) != NodeType_Node {\n")
	fmt.Fprintf(buf, "\t\t\treturn true\n")
	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\tswitch t.NameID(cid) {\n")

	for _, f := range si.Fields {
		switch f.Kind {
		case FieldText:
			fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
			fmt.Fprintf(buf, "\t\t\tout.%s = t.Text(cid)\n", f.GoName)
			fmt.Fprintf(buf, "\t\t\treturn false\n")

		case FieldNamedRule:
			fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
			fmt.Fprintf(buf, "\t\t\tval, err := Extract%s(t, cid)\n", f.GoType)
			fmt.Fprintf(buf, "\t\t\tif err == nil {\n")
			fmt.Fprintf(buf, "\t\t\t\tout.%s = val\n", f.GoName)
			fmt.Fprintf(buf, "\t\t\t}\n")
			fmt.Fprintf(buf, "\t\t\treturn false\n")

		case FieldSlice:
			fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
			fmt.Fprintf(buf, "\t\t\tval, err := Extract%s(t, cid)\n", f.ElemType)
			fmt.Fprintf(buf, "\t\t\tif err == nil {\n")
			fmt.Fprintf(buf, "\t\t\t\tout.%s = append(out.%s, val)\n", f.GoName, f.GoName)
			fmt.Fprintf(buf, "\t\t\t}\n")
			fmt.Fprintf(buf, "\t\t\treturn false\n")

		case FieldOptional:
			innerType := strings.TrimPrefix(f.GoType, "*")
			fmt.Fprintf(buf, "\t\tcase _nameID_%s:\n", f.LLTag)
			if innerType == "string" {
				fmt.Fprintf(buf, "\t\t\ts := t.Text(cid)\n")
				fmt.Fprintf(buf, "\t\t\tout.%s = &s\n", f.GoName)
			} else {
				fmt.Fprintf(buf, "\t\t\tval, err := Extract%s(t, cid)\n", innerType)
				fmt.Fprintf(buf, "\t\t\tif err == nil {\n")
				fmt.Fprintf(buf, "\t\t\t\tout.%s = &val\n", f.GoName)
				fmt.Fprintf(buf, "\t\t\t}\n")
			}
			fmt.Fprintf(buf, "\t\t\treturn false\n")
		}
	}

	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\treturn true\n")
	fmt.Fprintf(buf, "\t})\n")
}
