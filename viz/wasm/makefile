build:
	cp "$(shell go env GOROOT)/misc/wasm/wasm_exec.js" src/wasm_exec.js
	cd lib && GOOS=js GOARCH=wasm go build -o ../src/langlang.wasm . && cd ..
	cp src/langlang.wasm dist/langlang.wasm

clean:
	rm -f src/langlang.wasm
	rm -f src/wasm_exec.js
	rm -f dist/langlang.wasm

tidy:
	cd lib && go mod tidy && cd ..

.PHONY: build run clean