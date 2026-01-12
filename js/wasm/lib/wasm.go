//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	langlang "github.com/clarete/langlang/go"
)

var (
	nextMatcherID uint32 = 1

	matchers = map[uint32]langlang.Matcher{}
)

func register() {
	obj := js.Global().Get("Object").New()

	// constants
	nodeTypes := js.Global().Get("Object").New()
	nodeTypes.Set("String", int(langlang.NodeType_String))
	nodeTypes.Set("Sequence", int(langlang.NodeType_Sequence))
	nodeTypes.Set("Node", int(langlang.NodeType_Node))
	nodeTypes.Set("Error", int(langlang.NodeType_Error))
	obj.Set("NodeType", nodeTypes)

	obj.Set("matcherFromString", keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return jsErr("matcherFromString(grammar: string, cfg?: object): missing grammar")
		}
		grammar := args[0].String()

		cfg := langlang.NewConfig()
		if len(args) >= 2 && args[1].Type() == js.TypeObject {
			if err := applyConfig(cfg, args[1]); err != nil {
				return jsErr(err.Error())
			}
		}
		loader := langlang.NewInMemoryImportLoader()
		entry := "grammar.peg"
		loader.Add(entry, []byte(grammar))
		resolver := langlang.NewImportResolver(loader)
		m, err := resolver.MatcherFor(entry, cfg)
		if err != nil {
			return jsErr(err.Error())
		}
		id := nextMatcherID
		nextMatcherID++
		matchers[id] = m
		r := js.Global().Get("Object").New()
		r.Set("id", int(id))
		return jsOK(r)
	})))

	// matcherFromFiles compiles a grammar from an in-memory "filesystem".
	//
	// JS signature:
	//   matcherFromFiles(entry: string, files: Array<{path:string, content:string}> | Record<string,string>, cfg?: object)
	obj.Set("matcherFromFiles", keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 {
			return jsErr("matcherFromFiles(entry: string, files: Array<{path:string, content:string}> | Record<string,string>, cfg?: object): missing args")
		}
		entry := args[0].String()
		files := args[1]

		cfg := langlang.NewConfig()
		if len(args) >= 3 && args[2].Type() == js.TypeObject {
			if err := applyConfig(cfg, args[2]); err != nil {
				return jsErr(err.Error())
			}
		}

		loader := langlang.NewInMemoryImportLoader()

		// Accept either:
		// - Array<{path, content}>
		// - Record<string, string>
		arrCtor := js.Global().Get("Array")
		if files.Type() == js.TypeObject && files.InstanceOf(arrCtor) {
			for i := 0; i < files.Length(); i++ {
				item := files.Index(i)
				if item.Type() != js.TypeObject {
					return jsErr("matcherFromFiles: files array items must be objects {path, content}")
				}
				pathVal := item.Get("path")
				contentVal := item.Get("content")
				if pathVal.Type() != js.TypeString || contentVal.Type() != js.TypeString {
					return jsErr("matcherFromFiles: files array items must have string {path, content}")
				}
				loader.Add(pathVal.String(), []byte(contentVal.String()))
			}
		} else {
			return jsErr("matcherFromFiles: files must be an array of SourceFile objects")
		}

		resolver := langlang.NewImportResolver(loader)
		m, err := resolver.MatcherFor(entry, cfg)
		if err != nil {
			return jsErr(err.Error())
		}
		id := nextMatcherID
		nextMatcherID++
		matchers[id] = m
		r := js.Global().Get("Object").New()
		r.Set("id", int(id))
		return jsOK(r)
	})))

	obj.Set("freeMatcher", keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return jsErr("freeMatcher(id): missing id")
		}
		id := uint32(args[0].Int())
		delete(matchers, id)
		return jsOK(js.Undefined())
	})))

	obj.Set("match", keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 {
			return jsErr("match(matcherId, input: string): missing args")
		}
		mid := uint32(args[0].Int())
		m, ok := matchers[mid]
		if !ok {
			return jsErr(fmt.Sprintf("unknown matcher id %d", mid))
		}
		input := []byte(args[1].String())
		t, consumed, err := m.Match(input)

		resp := js.Global().Get("Object").New()
		resp.Set("consumed", consumed)
		resp.Set("value", matchToValue(t, err))

		return jsOK(resp)
	})))

	js.Global().Set("langlang", obj)

	// If JS installed a readiness resolver, notify it now that the API is registered.
	// This avoids polling for globalThis.langlang.
	if ready := js.Global().Get("__langlangReadyResolve"); ready.Type() == js.TypeFunction {
		ready.Invoke()
		js.Global().Set("__langlangReadyResolve", js.Undefined())
	}
}

func applyConfig(cfg *langlang.Config, opts js.Value) error {
	if opts.IsNull() || opts.IsUndefined() || opts.Type() != js.TypeObject {
		return nil
	}
	keys := js.Global().Get("Object").Call("keys", opts)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		v := opts.Get(k)
		switch v.Type() {
		case js.TypeBoolean:
			cfg.SetBool(k, v.Bool())
		case js.TypeNumber:
			cfg.SetInt(k, v.Int())
		case js.TypeString:
			cfg.SetString(k, v.String())
		default:
			return fmt.Errorf("unsupported config value type for %q", k)
		}
	}
	return nil
}

func emptySeq() js.Value {
	o := js.Global().Get("Object").New()
	o.Set("type", "sequence")
	o.Set("count", 0)
	o.Set("items", js.Global().Get("Array").New(0))
	return o
}

func matchToValue(t langlang.Tree, err error) js.Value {
	if err != nil {
		o := js.Global().Get("Object").New()
		o.Set("type", "error")
		o.Set("message", err.Error())
		if t != nil {
			if root, ok := t.Root(); ok {
				o.Set("expr", treeNodeToValue(t, root))
			}
		}
		return o
	}

	if t == nil {
		return emptySeq()
	}

	root, ok := t.Root()
	if !ok {
		return emptySeq()
	}

	return treeNodeToValue(t, root)
}

func treeNodeToValue(t langlang.Tree, id langlang.NodeID) js.Value {
	o := js.Global().Get("Object").New()
	o.Set("span", jsSpan(t, t.Span(id)))

	switch t.Type(id) {
	case langlang.NodeType_String:
		o.Set("type", "string")
		o.Set("value", t.Text(id))
		return o
	case langlang.NodeType_Sequence:
		kids := t.Children(id)
		arr := js.Global().Get("Array").New(len(kids))
		for i, kid := range kids {
			arr.SetIndex(i, treeNodeToValue(t, kid))
		}
		o.Set("type", "sequence")
		o.Set("count", len(kids))
		o.Set("items", arr)
		return o
	case langlang.NodeType_Node:
		o.Set("type", "node")
		o.Set("name", t.Name(id))
		if child, ok := t.Child(id); ok {
			o.Set("expr", treeNodeToValue(t, child))
		} else {
			o.Set("expr", emptySeq())
		}
		return o
	case langlang.NodeType_Error:
		o.Set("type", "error")
		o.Set("label", t.Name(id))
		if child, ok := t.Child(id); ok {
			o.Set("expr", treeNodeToValue(t, child))
		}
		return o
	default:
		o.Set("type", "error")
		o.Set("message", fmt.Sprintf("unknown node type %d", t.Type(id)))
		return o
	}
}

func jsLocation(t langlang.Tree, location langlang.Location) js.Value {
	o := js.Global().Get("Object").New()
	o.Set("line", location.Line)
	o.Set("column", location.Column)
	o.Set("cursor", location.Cursor)
	o.Set("utf16Cursor", t.CursorU16(location.Cursor))
	return o
}

func jsSpan(t langlang.Tree, span langlang.Span) js.Value {
	o := js.Global().Get("Object").New()
	o.Set("start", jsLocation(t, span.Start))
	o.Set("end", jsLocation(t, span.End))
	return o
}

func jsOK(v any) js.Value {
	o := js.Global().Get("Object").New()
	o.Set("ok", true)
	o.Set("value", v)
	return o
}

func jsErr(msg string) js.Value {
	o := js.Global().Get("Object").New()
	o.Set("ok", false)
	o.Set("error", msg)
	return o
}
