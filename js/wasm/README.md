# @langlang/wasm

To install dependencies:

```bash
bun install
```

To run:

```bash
bun run src/index.ts
```

## Building the Go WASM binary (no Makefile needed)

This package uses `package.json` scripts (shell commands) that replace `make build/clean/tidy/copy`:

```bash
# Build src/langlang.wasm and copy it into dist/langlang.wasm
bun run wasm:build

# Remove generated wasm artifacts
bun run wasm:clean

# Run `go mod tidy` inside ./lib
bun run wasm:tidy

# Copy wasm_exec.js from Go's GOROOT into src/wasm_exec.js
bun run wasm:copy
```

## Idiomatic JS API (Matcher / Tree / Node)

After you initialize the WASM module (see `initializeLangLangWasm`), the Go side exposes low-level bindings on `globalThis.langlang`.
This package provides an ergonomic wrapper inspired by `go/wasm/langlang.mjs`:

```ts
import { initializeLangLangWasm } from "@langlang/wasm";
import wasmUrl from "@langlang/wasm/langlang.wasm?url";
import wasmExecUrl from "@langlang/wasm/wasm_exec.js?url";

const ll = await initializeLangLangWasm(wasmUrl, wasmExecUrl);

const matcher = ll.matcherFromBytes("Start <- 'a'+");
const { tree, error } = matcher.match("aaa");
matcher.dispose();

if (!error && tree) {
  console.log(tree.pretty());
  tree.dispose();
}
```

## Even simpler (browser / Vite): no URLs

If your bundler supports `?url` asset imports (Vite does), you can avoid passing URLs:

```ts
import { initializeLangLangWasmFromPackageAssets } from "@langlang/wasm/browser";

const ll = await initializeLangLangWasmFromPackageAssets();
```

This project was created using `bun init` in bun v1.2.10. [Bun](https://bun.sh) is a fast all-in-one JavaScript runtime.
