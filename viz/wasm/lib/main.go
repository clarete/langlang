package main

import (
	"syscall/js"
)

func main() {

	exposedObject := js.Global().Get("Object").New()

	exposedObject.Set("compileAndMatch", js.FuncOf(compileAndMatch))
	exposedObject.Set("compileJson", js.FuncOf(compileJson))

	js.Global().Set("langlangWasm", exposedObject)

	select {}
}
