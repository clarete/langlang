package langlang

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

//go:embed c/vm.c c/vm.h c/tree.c c/tree.h
var cEvalContent embed.FS

type GenCOptions struct {
	ParserName string
	HeaderPath string
	RemoveLib  bool
}

func GenCEval(asm *Program, opt GenCOptions) (string, error) {
	csrc, _, err := GenCEvalWithHeader(asm, opt)
	return csrc, err
}

func GenCEvalWithHeader(asm *Program, opt GenCOptions) (csrc string, hsrc string, err error) {
	if opt.ParserName == "" {
		opt.ParserName = "Parser"
	}
	g := newCEvalEmitter(opt)
	g.writePrelude()
	g.writeRuntime()
	g.writeParserProgram(Encode(asm))
	g.writeParserStruct()
	g.writeParserConstructor()
	g.writeParserMethods(asm)
	csrc, err = g.output()
	if err != nil {
		return "", "", err
	}

	// Collect Parse<Rule> names for header.
	addrs := make([]int, 0, len(asm.identifiers))
	for addr := range asm.identifiers {
		addrs = append(addrs, addr)
	}
	sort.Ints(addrs)
	ruleNames := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		strID := asm.identifiers[addr]
		ruleNames = append(ruleNames, sanitizeCIdent(asm.strings[strID]))
	}

	h := newCEvalHeaderEmitter(opt, ruleNames)
	h.writeHeader()
	hsrc, err = h.output()
	if err != nil {
		return "", "", err
	}
	return csrc, hsrc, nil
}

type cEvalEmitter struct {
	options GenCOptions
	out     *outputWriter
}

func newCEvalEmitter(opt GenCOptions) *cEvalEmitter {
	return &cEvalEmitter{
		options: opt,
		out:     newOutputWriter("  "),
	}
}

func (g *cEvalEmitter) output() (string, error) {
	return g.out.buffer.String(), nil
}

func (g *cEvalEmitter) writePrelude() {
	g.out.writel("/*")
	g.out.writel(" * Auto-generated C parser by langlang.")
	g.out.writel(" *")
	g.out.writel(" * This file embeds the runtime unless RemoveLib=true.")
	g.out.writel(" */")
	g.out.writel("")
	g.out.writel(fmt.Sprintf(`#include "%s"`, g.options.HeaderPath))
	g.out.writel("")
	g.out.writel("/* This tells the embedded runtime sources (c/vm.c, c/tree.c) to not include headers. */")
	g.out.writel("#define LANGLANG_EMBEDDED 1")
	g.out.writel("")
}

func (g *cEvalEmitter) writeRuntime() {
	if g.options.RemoveLib {
		return
	}
	for _, file := range []string{"c/vm.c", "c/tree.c"} {
		data, err := cEvalContent.ReadFile(file)
		if err != nil {
			panic(err.Error())
		}
		g.out.writel(fmt.Sprintf("/* ---- BEGIN embedded runtime: %s ---- */", file))
		g.out.writel(string(data))
		g.out.writel(fmt.Sprintf("/* ---- END embedded runtime: %s ---- */", file))
		g.out.writel("")
	}

	g.out.writel("/* ---- BEGIN generated parser ---- */")
	g.out.writel("")
}

func (g *cEvalEmitter) writeParserProgram(bt *Bytecode) {
	pn := sanitizeCIdent(g.options.ParserName)

	// Bytecode arrays
	g.out.writel("/* Bytecode for generated parser */")

	g.out.writeil(fmt.Sprintf("static const uint8_t %s_bytecode_code[] = {", pn))
	g.out.indent()
	g.out.writei("")
	for i, b := range bt.code {
		if i > 0 && i%24 == 0 {
			g.out.writel("")
			g.out.writei("")
		}
		g.out.write(fmt.Sprintf("%d,", b))
		g.out.write(" ")
	}
	g.out.writel("")
	g.out.unindent()
	g.out.writel("};")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("static const char *%s_bytecode_strs[] = {", pn))
	g.out.indent()
	for _, s := range bt.strs {
		g.out.writeil(fmt.Sprintf("\"%s\",", escapeLiteral(s)))
	}
	g.out.unindent()
	g.out.writel("};")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("static const ll_charset %s_bytecode_sets[] = {", pn))
	g.out.indent()
	for _, set := range bt.sets {
		g.out.writei("{ .bits = {")
		for i := 0; i < len(set.bits); i++ {
			g.out.write(fmt.Sprintf("%d,", set.bits[i]))
		}
		g.out.writel("} },")
	}
	g.out.unindent()
	g.out.writel("};")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("static const ll_bytecode %s_bytecode = {", pn))
	g.out.indent()
	g.out.writeil(fmt.Sprintf(".code = (uint8_t *)&%s_bytecode_code,", pn))
	g.out.writeil(fmt.Sprintf(".code_len = %d,", len(bt.code)))
	g.out.writeil(fmt.Sprintf(".strs = (const char **)&%s_bytecode_strs,", pn))
	g.out.writeil(fmt.Sprintf(".strs_len = %d,", len(bt.strs)))
	g.out.writeil(fmt.Sprintf(".sets = (ll_charset *)&%s_bytecode_sets,", pn))
	g.out.writeil(fmt.Sprintf(".sets_len = %d,", len(bt.sets)))
	g.out.unindent()
	g.out.writeil("};")
	g.out.writel("")

	// Init function (heap-owns code/strs pointer-array/sets so ll_bytecode_free is safe)
	g.out.writeil(fmt.Sprintf("static void init_bytecode_for_%s(ll_bytecode *bc) {", pn))
	g.out.indent()
	// g.out.writeil("ll_bytecode_init(bc);")
	// g.out.writel("")

	// rxps
	xps := make([]int, 0, len(bt.rxps))
	for xp := range bt.rxps {
		xps = append(xps, xp)
	}
	sort.Ints(xps)
	g.out.writeil("/* Recovery Expressions Map */")
	for _, k := range xps {
		v := bt.rxps[k]
		g.out.writeil(fmt.Sprintf("ll_i2i_map_put(&bc->rxps, %d, %d);", k, v))
	}
	g.out.writel("")

	// rxbs
	g.out.writeil("/* Recovery Expressions Bitset */")
	for i := 0; i < len(bt.rxbs); i++ {
		g.out.writeil(fmt.Sprintf("bc->rxbs.w[%d] = (uint64_t)%d;", i, bt.rxbs[i]))
	}
	g.out.writel("")

	// smap
	g.out.writeil("/* strings map (smap) */")
	keys := make([]string, 0, len(bt.smap))
	for k := range bt.smap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := bt.smap[k]
		g.out.writeil(fmt.Sprintf("ll_s2i_map_put(&bc->smap, %s, %d);", strconv.Quote(k), v))
	}
	g.out.writel("")

	g.out.writeil("/* Precompute expected sets for show-fails. */")
	g.out.writeil("//ll_bytecode_build_expected_sets(bc);")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")
}

func (g *cEvalEmitter) writeParserStruct() {
	pn := sanitizeCIdent(g.options.ParserName)
	// Opaque in the header; defined here.
	g.out.writeil(fmt.Sprintf("struct %s {", pn))
	g.out.indent()
	g.out.writeil("const uint8_t *input;")
	g.out.writeil("int input_len;")
	g.out.writeil("ll_bytecode bc;")
	g.out.writeil("ll_vm vm;")
	g.out.unindent()
	g.out.writel("};")
	g.out.writel(fmt.Sprintf("typedef struct %s %s;", pn, pn))
	g.out.writel("")
}

func (g *cEvalEmitter) writeParserConstructor() {
	pn := sanitizeCIdent(g.options.ParserName)

	g.out.writeil(fmt.Sprintf("void %s_SetInput(%s *p, const uint8_t *input, int input_len) {", pn, pn))
	g.out.indent()
	g.out.writeil("p->input = input;")
	g.out.writeil("p->input_len = input_len;")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("void %s_Init(%s *p) {", pn, pn))
	g.out.indent()

	g.out.writeil(fmt.Sprintf("ll_bytecode *bc = (ll_bytecode *)malloc(sizeof(%s_bytecode));;", pn))
	g.out.writeil(fmt.Sprintf("memcpy(bc, (ll_bytecode*)&%s_bytecode, sizeof(%s_bytecode));", pn, pn))
	g.out.writeil(fmt.Sprintf("init_bytecode_for_%s(bc);", pn))
	g.out.writeil("memset(p, 0, sizeof(*p));")
	g.out.writeil("ll_vm_init(&p->vm, bc);")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("void %s_Free(%s *p) {", pn, pn))
	g.out.indent()
	g.out.writeil("ll_vm_free(&p->vm);")
	g.out.writeil("ll_bytecode_free(&p->bc);")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("void %s_SetShowFails(%s *p, bool v) {", pn, pn))
	g.out.indent()
	g.out.writeil("ll_vm_set_show_fails(&p->vm, v);")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("%s *%s_New(void) {", pn, pn))
	g.out.indent()
	g.out.writeil(fmt.Sprintf("%s *p = (%s *)malloc(sizeof(%s));", pn, pn, pn))
	g.out.writeil("if (!p) { fprintf(stderr, \"out of memory\\n\"); abort(); }")
	g.out.writeil(fmt.Sprintf("%s_Init(p);", pn))
	g.out.writeil("return p;")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")

	g.out.writeil(fmt.Sprintf("void %s_Delete(%s *p) {", pn, pn))
	g.out.indent()
	g.out.writeil("if (!p) return;")
	g.out.writeil(fmt.Sprintf("%s_Free(p);", pn))
	g.out.writeil("free(p);")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")
}

func (g *cEvalEmitter) writeParserMethods(asm *Program) {
	pn := sanitizeCIdent(g.options.ParserName)

	// Map instruction-index label addresses to bytecode pc offsets (same as gen.go).
	cursor := 0
	addrmap := make(map[int]int, len(asm.identifiers))
	for i, instruction := range asm.code {
		switch instruction.(type) {
		case ILabel:
			addrmap[i] = cursor
		default:
			cursor += instruction.SizeInBytes()
		}
	}

	// Generic parseFn
	g.out.writeil(fmt.Sprintf("static ll_tree *%s_parse_fn(%s *p, int addr, int *out_cursor, ll_parsing_error *out_err) {", pn, pn))
	g.out.indent()
	g.out.writeil("return ll_vm_match_rule(&p->vm, p->input, p->input_len, addr, out_cursor, out_err);")
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")

	// ParseXXX methods
	addrs := make([]int, 0, len(asm.identifiers))
	for addr := range asm.identifiers {
		addrs = append(addrs, addr)
	}
	sort.Ints(addrs)
	for _, addr := range addrs {
		strID := asm.identifiers[addr]
		name := sanitizeCIdent(asm.strings[strID])
		pc := addrmap[addr]
		g.out.writeil(fmt.Sprintf("ll_tree *%s_Parse%s(%s *p, int *out_cursor, ll_parsing_error *out_err) {", pn, name, pn))
		g.out.indent()
		g.out.writeil(fmt.Sprintf("return %s_parse_fn(p, %d, out_cursor, out_err);", pn, pc))
		g.out.unindent()
		g.out.writel("}")
	}
	g.out.writel("")

	// Parse() entrypoint matches gen.go convention: addr 5
	g.out.writeil(fmt.Sprintf("ll_tree *%s_Parse(%s *p, int *out_cursor, ll_parsing_error *out_err) {", pn, pn))
	g.out.indent()
	g.out.writeil(fmt.Sprintf("return %s_parse_fn(p, 5, out_cursor, out_err);", pn))
	g.out.unindent()
	g.out.writel("}")
	g.out.writel("")
}

func sanitizeCIdent(s string) string {
	if s == "" {
		return "X"
	}
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		if i == 0 {
			if r == '_' || unicode.IsLetter(r) {
				b.WriteRune(r)
				continue
			}
			if unicode.IsDigit(r) {
				b.WriteRune('_')
				b.WriteRune(r)
				continue
			}
			b.WriteRune('_')
			continue
		}
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// ---- Header generator ----

type cEvalHeaderEmitter struct {
	options GenCOptions
	rules   []string
	out     *outputWriter
}

func newCEvalHeaderEmitter(opt GenCOptions, rules []string) *cEvalHeaderEmitter {
	return &cEvalHeaderEmitter{
		options: opt,
		rules:   rules,
		out:     newOutputWriter("  "),
	}
}

func (h *cEvalHeaderEmitter) output() (string, error) {
	return h.out.buffer.String(), nil
}

func (h *cEvalHeaderEmitter) writeHeader() {
	pn := sanitizeCIdent(h.options.ParserName)
	guard := strings.ToUpper(pn) + "_H"

	h.out.writel("/* Auto-generated C parser header by langlang */")
	h.out.writel(fmt.Sprintf("#ifndef %s", guard))
	h.out.writel(fmt.Sprintf("#define %s", guard))
	h.out.writel("")
	h.out.writel("#include <stdbool.h>")
	h.out.writel("#include <stdint.h>")
	h.out.writel("")

	h.out.writel("#include <stdint.h>")

	for _, file := range []string{"c/tree.h", "c/vm.h"} {
		data, err := cEvalContent.ReadFile(file)
		if err != nil {
			panic(err.Error())
		}
		h.out.writel(fmt.Sprintf("/* ---- BEGIN embedded runtime: %s ---- */", file))
		h.out.writel(string(data))
		h.out.writel(fmt.Sprintf("/* ---- END embedded runtime: %s ---- */", file))
		h.out.writel("")
	}

	// // Minimal tree-facing API (opaque type + a few helpers).
	// h.out.writel("typedef int32_t ll_node_id;")
	// h.out.writel("typedef struct ll_tree ll_tree;")
	// h.out.writel("")
	// h.out.writel("void ll_tree_free(ll_tree *t);")
	// h.out.writel("bool ll_tree_root(const ll_tree *t, ll_node_id *out_id);")
	// h.out.writel("char *ll_tree_name(const ll_tree *t, ll_node_id id);")
	// h.out.writel("char *ll_tree_pretty(const ll_tree *t, ll_node_id id);")
	// h.out.writel("char *ll_tree_highlight(const ll_tree *t, ll_node_id id);")
	// h.out.writel("char *ll_tree_text(const ll_tree *t, ll_node_id id);")
	// h.out.writel("ll_range ll_tree_range(const ll_tree *t, ll_node_id id);")
	// h.out.writel("")

	// // Mirror vm.c's parsing error struct (so users can consume/free error messages).
	// h.out.writel("")

	// Opaque parser type.
	h.out.writel(fmt.Sprintf("typedef struct %s %s;", pn, pn))
	h.out.writel("")

	// Heap lifecycle helpers.
	h.out.writel(fmt.Sprintf("%s *%s_New(void);", pn, pn))
	h.out.writel(fmt.Sprintf("void %s_Delete(%s *p);", pn, pn))
	h.out.writel("")

	// Stack-friendly lifecycle helpers (optional to use).
	h.out.writel(fmt.Sprintf("void %s_Init(%s *p);", pn, pn))
	h.out.writel(fmt.Sprintf("void %s_Free(%s *p);", pn, pn))
	h.out.writel("")

	// Input + parse entrypoints.
	h.out.writel(fmt.Sprintf("void %s_SetInput(%s *p, const uint8_t *input, int input_len);", pn, pn))
	h.out.writel(fmt.Sprintf("void %s_SetShowFails(%s *p, bool v);", pn, pn))
	h.out.writel(fmt.Sprintf("ll_tree *%s_Parse(%s *p, int *out_cursor, ll_parsing_error *out_err);", pn, pn))
	h.out.writel("")

	if len(h.rules) > 0 {
		h.out.writel("/* Parse<Rule> entrypoints */")
		for _, r := range h.rules {
			h.out.writel(fmt.Sprintf("ll_tree *%s_Parse%s(%s *p, int *out_cursor, ll_parsing_error *out_err);", pn, r, pn))
		}
		h.out.writel("")
	}
	h.out.writel(fmt.Sprintf("#endif /* %s */", guard))
}
