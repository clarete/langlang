package treesitter_json

// #cgo CFLAGS: -std=c11 -fPIC
// #include "tree-sitter-json-grammar/src/parser.c"
// // Unnamed structs are not supported by cgo, so we need to give them names
import "C"

import (
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func GetJSONLanguage() *sitter.Language {
	return sitter.NewLanguage(unsafe.Pointer(C.tree_sitter_json()))
}

