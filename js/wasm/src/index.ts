import langlang from "./langlang";
export * from "./langlang";

// Minimal typing for the Go WASM runtime injected by `wasm_exec.js`.
// (We intentionally avoid DOM typings and any dependency on Go's upstream TS defs.)
type Go = {
    importObject: any;
    run(instance: any): Promise<void>;
};

export class GoLangError<T = Error> extends Error {
    constructor(message: string, cause?: T) {
        super(message);
        this.name = "GoLangError";
        this.cause = cause as Error;
    }
}

let initPromise: Promise<langlang> | null = null;

let goWasmRuntimePromise: Promise<void> | null = null;

async function ensureGoWasmRuntime(wasmExecUrl?: string) {
    const g = globalThis as any;

    // Already loaded (browser) or provided by some other environment.
    if (typeof g.Go === "function") return;

    // Nothing we can do without a URL to the runtime.
    if (!wasmExecUrl) return;

    // Prevent double-injecting the runtime when multiple callers race.
    if (goWasmRuntimePromise) return goWasmRuntimePromise;

    goWasmRuntimePromise = (async () => {
        // Use globalThis.document to avoid requiring DOM typings at build time.
        const doc = g.document;
        if (!doc?.createElement || !doc?.head?.appendChild) {
            // Not a browser-like environment; leave it to the existing error below.
            return;
        }

        // If something else loaded it while we were waiting, skip injection.
        if (typeof g.Go === "function") return;

        await new Promise<void>((resolve, reject) => {
            const script = doc.createElement("script");
            script.src = wasmExecUrl;
            script.async = true;
            script.onload = () => resolve();
            script.onerror = () =>
                reject(new Error(`Failed to load ${wasmExecUrl}`));
            doc.head.appendChild(script);
        });
    })();

    try {
        await goWasmRuntimePromise;
    } finally {
        // Keep the promise around if it succeeded (Go runtime is global and persistent),
        // but allow retry if it failed.
        if (typeof (globalThis as any).Go !== "function") {
            goWasmRuntimePromise = null;
        }
    }
}

/**
 * initializeLangLangWasm boots the Go WASM runtime (via wasm_exec.js) and returns a
 * JS wrapper that offers convenience helpers (compileAndMatch/compileJson) on top
 * of the low-level api.go-aligned bindings exported from Go.
 *
 * This init is one-shot; repeated calls reuse the same running runtime.
 */
export async function initializeLangLangWasm(
    langlangBinUrl: string,
    wasmExecUrl?: string,
) {
    if (initPromise) return initPromise;

    initPromise = (async () => {
        await ensureGoWasmRuntime(wasmExecUrl);
        if (typeof (globalThis as any).Go !== "function") {
            throw new GoLangError(
                "Go WASM runtime not found. Did you load wasm_exec.js first?",
            );
        }

        const go = new (globalThis as any).Go() as Go;

        let readyResolve: (() => void) | undefined;
        const ready = new Promise<void>((resolve) => (readyResolve = resolve));
        (globalThis as any).__langlangReadyResolve = () => readyResolve?.();

        const wasmRsp = await fetch(langlangBinUrl);
        if (!wasmRsp.ok) {
            throw new GoLangError("Failed to fetch wasm file");
        }

        const webAssembly = await WebAssembly.instantiateStreaming(
            wasmRsp,
            go.importObject,
        );
        // NOTE: go.run does not resolve until the Go program exits.
        go.run(webAssembly.instance);

        // Wait for Go to signal readiness (see viz/wasm/lib/langlang_js.go).
        await Promise.race([
            ready,
            new Promise((_, reject) =>
                setTimeout(
                    () =>
                        reject(
                            new GoLangError(
                                "timeout waiting for langlang wasm init",
                            ),
                        ),
                    2000,
                ),
            ),
        ]);

        // Go exports the API under both names, but we prefer the new one.
        const raw =
            (globalThis as any).langlang ?? (globalThis as any).langlangWasm;
        if (!raw)
            throw new GoLangError("langlang wasm bindings not initialized");

        return new langlang(raw);
    })();

    return initPromise;
}

export * from "./types";
