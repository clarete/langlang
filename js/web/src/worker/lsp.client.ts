/**
 * LSP Worker Client - Main-thread interface to the LSP worker.
 *
 * Provides an async API for LSP operations with automatic request/response
 * correlation and server notification handling.
 */

import type { WorkerRequest, WorkerResponse } from "./lsp.worker";

export type LspMessage = {
    jsonrpc: "2.0";
    id?: number | string;
    method?: string;
    params?: unknown;
    result?: unknown;
    error?: { code: number; message: string };
};

export type LspNotificationHandler = (method: string, params: any) => void;

export type LspClientConfig = {
    wasmUrl: string;
    wasmExecUrl: string;
    /** Called when the server sends a notification (e.g., publishDiagnostics) */
    onNotification?: LspNotificationHandler;
    /** Called when the worker is ready */
    onReady?: () => void;
    /** Called when there's an error during initialization */
    onError?: (error: Error) => void;
    /** Enable debug logging */
    debug?: boolean;
};

type PendingRequest = {
    resolve: (result: any) => void;
    reject: (error: Error) => void;
};

/**
 * Creates an LSP client that communicates with a worker-based LSP server.
 */
export function createLspWorkerClient(config: LspClientConfig) {
    const { wasmUrl, wasmExecUrl, onNotification, onReady, onError, debug } = config;

    let worker: Worker | null = null;
    let nextRequestId = 1;
    const pending = new Map<number, PendingRequest>();
    let ready = false;
    let readyPromise: Promise<void>;
    let readyResolve: (() => void) | undefined;
    let readyReject: ((error: Error) => void) | undefined;

    const dbg = (...args: any[]) => {
        if (!debug) return;
        console.log("[lsp-client]", ...args);
    };

    // Create the ready promise
    readyPromise = new Promise<void>((resolve, reject) => {
        readyResolve = resolve;
        readyReject = reject;
    });

    // Handle messages from the worker
    function handleWorkerMessage(event: MessageEvent<WorkerResponse>) {
        const msg = event.data;
        dbg("recv", msg);

        switch (msg.type) {
            case "ready":
                ready = true;
                readyResolve?.();
                onReady?.();
                break;

            case "init-error":
                const initError = new Error(msg.error);
                readyReject?.(initError);
                onError?.(initError);
                break;

            case "lsp-response":
                handleLspResponse(msg.id, msg.messages);
                break;

            case "lsp-error": {
                const req = pending.get(msg.id);
                if (req) {
                    pending.delete(msg.id);
                    req.reject(new Error(msg.error));
                }
                break;
            }

            case "compile-response": {
                const req = pending.get(msg.id);
                if (req) {
                    pending.delete(msg.id);
                    req.resolve({ matcherId: msg.matcherId });
                }
                break;
            }

            case "compile-error": {
                const req = pending.get(msg.id);
                if (req) {
                    pending.delete(msg.id);
                    req.reject(new Error(msg.error));
                }
                break;
            }

            case "match-response": {
                const req = pending.get(msg.id);
                if (req) {
                    pending.delete(msg.id);
                    req.resolve({ consumed: msg.consumed, value: msg.value });
                }
                break;
            }

            case "match-error": {
                const req = pending.get(msg.id);
                if (req) {
                    pending.delete(msg.id);
                    req.reject(new Error(msg.error));
                }
                break;
            }
        }
    }

    function handleLspResponse(requestId: number, messages: any[]) {
        // The worker returns an array of messages.
        // We need to separate responses (with matching id) from notifications.
        let foundResponse = false;
        
        for (const msg of messages) {
            if (msg.id !== undefined) {
                // This is a response to a request
                foundResponse = true;
                const req = pending.get(requestId);
                if (req) {
                    pending.delete(requestId);
                    if (msg.error) {
                        req.reject(new Error(msg.error.message || "LSP error"));
                    } else {
                        req.resolve(msg.result);
                    }
                }
            } else if (msg.method) {
                // This is a notification from the server
                dbg("notification", msg.method, msg.params);
                onNotification?.(msg.method, msg.params);
            }
        }
        
        // If no response was found, this was a notification from client to server.
        // Resolve the pending promise since notifications don't expect responses.
        if (!foundResponse) {
            const req = pending.get(requestId);
            if (req) {
                pending.delete(requestId);
                req.resolve(undefined);
            }
        }
    }

    function postMessage(msg: WorkerRequest) {
        if (!worker) {
            throw new Error("Worker not initialized");
        }
        dbg("send", msg);
        worker.postMessage(msg);
    }

    // Initialize the worker
    function start(): Promise<void> {
        if (worker) {
            return readyPromise;
        }

        // Create a classic worker (not module) so we can use importScripts()
        // for loading wasm_exec.js safely without eval()
        worker = new Worker(
            new URL("./lsp.worker.ts", import.meta.url),
            { type: "classic" }
        );
        worker.onmessage = handleWorkerMessage;
        worker.onerror = (e) => {
            const error = new Error(`Worker error: ${e.message}`);
            readyReject?.(error);
            onError?.(error);
        };

        // Send init message
        postMessage({ type: "init", wasmUrl, wasmExecUrl });

        return readyPromise;
    }

    // LSP request helper
    async function request(method: string, params: any): Promise<any> {
        await readyPromise;
        const id = nextRequestId++;
        const lspMessage: LspMessage = {
            jsonrpc: "2.0",
            id,
            method,
            params,
        };

        return new Promise((resolve, reject) => {
            pending.set(id, { resolve, reject });
            postMessage({ type: "lsp", id, message: lspMessage });
        });
    }

    // LSP notification helper (no response expected)
    async function notify(method: string, params: any): Promise<void> {
        await readyPromise;
        const id = nextRequestId++; // Still need an id for worker correlation
        const lspMessage: LspMessage = {
            jsonrpc: "2.0",
            // No id in the LSP message = notification
            method,
            params,
        };

        // For notifications, we don't wait for a response, but we still send
        // through the worker and may receive server notifications back
        return new Promise((resolve, reject) => {
            pending.set(id, {
                resolve: () => resolve(),
                reject,
            });
            postMessage({ type: "lsp", id, message: lspMessage });
        });
    }

    // Grammar compilation (also runs in worker)
    async function compile(
        grammar: string,
        config?: Record<string, any>
    ): Promise<{ matcherId: number }> {
        await readyPromise;
        const id = nextRequestId++;

        return new Promise((resolve, reject) => {
            pending.set(id, { resolve, reject });
            postMessage({ type: "compile", id, grammar, config });
        });
    }

    async function compileFiles(
        entry: string,
        files: Array<{ path: string; content: string }>,
        config?: Record<string, any>
    ): Promise<{ matcherId: number }> {
        await readyPromise;
        const id = nextRequestId++;

        return new Promise((resolve, reject) => {
            pending.set(id, { resolve, reject });
            postMessage({ type: "compile", id, grammar: "", files, entry, config });
        });
    }

    async function match(
        matcherId: number,
        input: string
    ): Promise<{ consumed: number; value: any }> {
        await readyPromise;
        const id = nextRequestId++;

        return new Promise((resolve, reject) => {
            pending.set(id, { resolve, reject });
            postMessage({ type: "match", id, matcherId, input });
        });
    }

    function freeMatcher(matcherId: number): void {
        if (!worker) return;
        postMessage({ type: "freeMatcher", matcherId });
    }

    function terminate(): void {
        if (worker) {
            worker.terminate();
            worker = null;
            ready = false;
        }
    }

    return {
        /** Start the worker and initialize WASM */
        start,
        /** Wait for the worker to be ready */
        get ready() {
            return readyPromise;
        },
        /** Check if the worker is ready */
        get isReady() {
            return ready;
        },
        /** Send an LSP request and wait for response */
        request,
        /** Send an LSP notification */
        notify,
        /** Compile a grammar (returns matcher id) */
        compile,
        /** Compile a grammar from multiple files */
        compileFiles,
        /** Run a match operation */
        match,
        /** Free a matcher */
        freeMatcher,
        /** Terminate the worker */
        terminate,
    };
}

export type LspWorkerClient = ReturnType<typeof createLspWorkerClient>;

