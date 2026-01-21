/**
 * LSP Worker - Runs the langlang WASM module in a dedicated worker thread.
 *
 * This keeps the main thread free for UI interactions while handling
 * LSP operations (diagnostics, completions, hover, etc.) asynchronously.
 *
 * NOTE: This worker runs as a "classic" worker (not module) to use importScripts()
 * for loading wasm_exec.js safely without eval().
 */

// Type declarations for the worker context
declare const self: {
    Go: new () => Go;
    langlang?: RawApi;
    langlangWasm?: RawApi;
    __langlangReadyResolve?: () => void;
    postMessage(message: WorkerResponse): void;
    onmessage: ((event: MessageEvent<WorkerRequest>) => void) | null;
};

declare function importScripts(...urls: string[]): void;

// Type for Go WASM runtime
interface Go {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
}

type RawResult<T> = { ok: true; value: T } | { ok: false; error: string };
type LspHandle = (message: string) => RawResult<string>;

interface RawApi {
    NodeType: {
        String: number;
        Sequence: number;
        Node: number;
        Error: number;
    };
    matcherFromString(grammar: string, cfg?: Record<string, unknown>): RawResult<{ id: number }>;
    matcherFromFiles(
        entry: string,
        files: Array<{ path: string; content: string }>,
        cfg?: Record<string, unknown>
    ): RawResult<{ id: number }>;
    freeMatcher(id: number): RawResult<void>;
    match(id: number, input: string): RawResult<{ consumed: number; value: unknown }>;
    lspHandle?: LspHandle;
}

// Message types for worker communication
export type WorkerRequest =
    | { type: "init"; wasmUrl: string; wasmExecUrl: string }
    | { type: "lsp"; id: number; message: unknown }
    | { type: "compile"; id: number; grammar: string; files?: Array<{ path: string; content: string }>; entry?: string; config?: Record<string, unknown> }
    | { type: "match"; id: number; matcherId: number; input: string }
    | { type: "freeMatcher"; matcherId: number };

export type WorkerResponse =
    | { type: "ready" }
    | { type: "init-error"; error: string }
    | { type: "lsp-response"; id: number; messages: unknown[] }
    | { type: "lsp-error"; id: number; error: string }
    | { type: "compile-response"; id: number; matcherId: number }
    | { type: "compile-error"; id: number; error: string }
    | { type: "match-response"; id: number; consumed: number; value: unknown }
    | { type: "match-error"; id: number; error: string };

let lspHandle: LspHandle | null = null;
let raw: RawApi | null = null;

async function initWasm(wasmUrl: string, wasmExecUrl: string): Promise<void> {
    // Load the Go runtime using importScripts (safe, no eval needed)
    importScripts(wasmExecUrl);

    if (typeof self.Go !== "function") {
        throw new Error("Go WASM runtime not found after loading wasm_exec.js");
    }

    const go = new self.Go();

    // Set up ready signal
    let readyResolve: (() => void) | undefined;
    const ready = new Promise<void>((resolve) => (readyResolve = resolve));
    self.__langlangReadyResolve = () => readyResolve?.();

    // Fetch and instantiate WASM
    const wasmResponse = await fetch(wasmUrl);
    if (!wasmResponse.ok) {
        throw new Error(`Failed to fetch WASM: ${wasmResponse.statusText}`);
    }

    const webAssembly = await WebAssembly.instantiateStreaming(
        wasmResponse,
        go.importObject
    );

    // Start the Go runtime (runs until program exits)
    const runPromise = go.run(webAssembly.instance);
    runPromise.catch((e: unknown) => {
        console.error("[lsp.worker] Go runtime crashed:", e);
    });

    // Wait for Go to signal readiness
    await Promise.race([
        ready,
        runPromise.then(() => {
            throw new Error("Go runtime exited before ready");
        }),
        new Promise((_, reject) =>
            setTimeout(() => reject(new Error("Timeout waiting for WASM init")), 5000)
        ),
    ]);

    // Store references to the raw API
    raw = self.langlang ?? self.langlangWasm ?? null;
    if (!raw) {
        throw new Error("langlang WASM bindings not initialized");
    }

    lspHandle = raw.lspHandle ?? null;
    if (!lspHandle) {
        console.warn("[lsp.worker] LSP handle not available in this WASM build");
    }
}

function handleLspMessage(id: number, message: unknown): void {
    if (!lspHandle) {
        self.postMessage({
            type: "lsp-error",
            id,
            error: "LSP not available",
        } satisfies WorkerResponse);
        return;
    }

    try {
        const result = lspHandle(JSON.stringify(message));
        if (!result.ok) {
            self.postMessage({
                type: "lsp-error",
                id,
                error: result.error,
            } satisfies WorkerResponse);
            return;
        }

        const messages = JSON.parse(result.value);
        self.postMessage({
            type: "lsp-response",
            id,
            messages: Array.isArray(messages) ? messages : [],
        } satisfies WorkerResponse);
    } catch (e) {
        self.postMessage({
            type: "lsp-error",
            id,
            error: e instanceof Error ? e.message : String(e),
        } satisfies WorkerResponse);
    }
}

function handleCompile(
    id: number,
    grammar: string,
    files?: Array<{ path: string; content: string }>,
    entry?: string,
    config?: Record<string, unknown>
): void {
    if (!raw) {
        self.postMessage({
            type: "compile-error",
            id,
            error: "WASM not initialized",
        } satisfies WorkerResponse);
        return;
    }

    try {
        let result: RawResult<{ id: number }>;
        if (files && entry) {
            result = raw.matcherFromFiles(entry, files, config);
        } else {
            result = raw.matcherFromString(grammar, config);
        }

        if (!result.ok) {
            self.postMessage({
                type: "compile-error",
                id,
                error: result.error,
            } satisfies WorkerResponse);
            return;
        }

        self.postMessage({
            type: "compile-response",
            id,
            matcherId: result.value.id,
        } satisfies WorkerResponse);
    } catch (e) {
        self.postMessage({
            type: "compile-error",
            id,
            error: e instanceof Error ? e.message : String(e),
        } satisfies WorkerResponse);
    }
}

function handleMatch(id: number, matcherId: number, input: string): void {
    if (!raw) {
        self.postMessage({
            type: "match-error",
            id,
            error: "WASM not initialized",
        } satisfies WorkerResponse);
        return;
    }

    try {
        const result = raw.match(matcherId, input);
        if (!result.ok) {
            self.postMessage({
                type: "match-error",
                id,
                error: result.error,
            } satisfies WorkerResponse);
            return;
        }

        self.postMessage({
            type: "match-response",
            id,
            consumed: result.value.consumed,
            value: result.value.value,
        } satisfies WorkerResponse);
    } catch (e) {
        self.postMessage({
            type: "match-error",
            id,
            error: e instanceof Error ? e.message : String(e),
        } satisfies WorkerResponse);
    }
}

function handleFreeMatcher(matcherId: number): void {
    if (!raw) return;
    try {
        raw.freeMatcher(matcherId);
    } catch (e) {
        console.error("[lsp.worker] Failed to free matcher:", e);
    }
}

// Message handler
self.onmessage = async (event: MessageEvent<WorkerRequest>) => {
    const msg = event.data;

    switch (msg.type) {
        case "init":
            try {
                await initWasm(msg.wasmUrl, msg.wasmExecUrl);
                self.postMessage({ type: "ready" } satisfies WorkerResponse);
            } catch (e) {
                self.postMessage({
                    type: "init-error",
                    error: e instanceof Error ? e.message : String(e),
                } satisfies WorkerResponse);
            }
            break;

        case "lsp":
            handleLspMessage(msg.id, msg.message);
            break;

        case "compile":
            handleCompile(msg.id, msg.grammar, msg.files, msg.entry, msg.config);
            break;

        case "match":
            handleMatch(msg.id, msg.matcherId, msg.input);
            break;

        case "freeMatcher":
            handleFreeMatcher(msg.matcherId);
            break;
    }
};
