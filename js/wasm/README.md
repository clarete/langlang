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

const matcher = ll.matcherFromString("Start <- 'a'+");
const { consumed, value } = matcher.match("aaa");
matcher.dispose();

console.log({ consumed, value });
```

## Multi-file grammars (imports) from memory

```ts
const matcher = ll.matcherFromFiles(
  "main.peg",
  [
    {
      path: "value.peg",
      content: `Value <- [0-9]+`,
    },
    {
      path: "main.peg",
      content: `@import Value from "./value.peg"
Main <- Value`,
    },
  ],
);

console.log(matcher.match("123").value);
matcher.dispose();
```

## Even simpler (browser / Vite): no URLs

If your bundler supports `?url` asset imports (Vite does), you can avoid passing URLs:

```ts
import { initializeLangLangWasmFromAssets } from "@langlang/wasm/browser";

const ll = await initializeLangLangWasmFromAssets();
```

This project was created using `bun init` in bun v1.2.10. [Bun](https://bun.sh) is a fast all-in-one JavaScript runtime.
