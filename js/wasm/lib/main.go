//go:build js && wasm

package main

import "syscall/js"

var keptFuncs []js.Func // prevent GC of exported callbacks

func keep(fn js.Func) js.Func {
	keptFuncs = append(keptFuncs, fn)
	return fn
}

func main() {
	register()
	// Keep the WASM runtime alive; JS calls into us via syscall/js callbacks.
	select {}
}
