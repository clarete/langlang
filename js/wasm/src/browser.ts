/**
 * Browser-first convenience initializer.
 *
 * This hides the `wasmUrl` + `wasmExecUrl` plumbing behind package assets, which
 * works great with bundlers like Vite that support `?url` imports.
 *
 * If your environment doesn't support `?url` (or you want custom hosting/CDN),
 * use `initializeLangLangWasm(wasmUrl, wasmExecUrl)` from the main entry instead.
 */
import { initializeLangLangWasm } from "./index";

// These are bundler (e.g. Vite) asset URL imports.
// @ts-expect-error - bundler-specific asset import
import wasmUrl from "./langlang.wasm?url";
// @ts-expect-error - bundler-specific asset import
import wasmExecUrl from "./wasm_exec.js?url";

export async function initializeLangLangWasmFromAssets() {
    return initializeLangLangWasm(wasmUrl, wasmExecUrl);
}
