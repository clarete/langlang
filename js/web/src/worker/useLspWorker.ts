/**
 * React hook for accessing the shared LSP worker client.
 *
 * This ensures the worker is only initialized once and provides
 * access to both LSP features and grammar compilation/matching.
 */

import { useEffect, useState, useSyncExternalStore } from "react";
import { createLspWorkerClient, type LspWorkerClient, type LspNotificationHandler } from "./lsp.client";

// Asset URLs for WASM - these use Vite's ?url import syntax
import wasmUrl from "@langlang/wasm/langlang.wasm?url";
import wasmExecUrl from "@langlang/wasm/wasm_exec.js?url";

// Singleton worker client
let sharedClient: LspWorkerClient | null = null;
let initPromise: Promise<void> | null = null;
let clientReady = false;
const listeners = new Set<() => void>();

function isDebugEnabled(): boolean {
    try {
        return (
            (globalThis as any).__llLspDebug === true ||
            window.localStorage?.getItem("ll_lsp_debug") === "1"
        );
    } catch {
        return false;
    }
}

type NotificationSubscriber = {
    method: string;
    handler: LspNotificationHandler;
};

const notificationSubscribers = new Set<NotificationSubscriber>();

function handleNotification(method: string, params: any) {
    for (const sub of notificationSubscribers) {
        if (sub.method === method || sub.method === "*") {
            sub.handler(method, params);
        }
    }
}

/**
 * Get or create the shared worker client.
 * This is safe to call multiple times - it will only create one worker.
 */
export function getSharedWorkerClient(): LspWorkerClient {
    if (!sharedClient) {
        sharedClient = createLspWorkerClient({
            wasmUrl,
            wasmExecUrl,
            onNotification: handleNotification,
            onReady: () => {
                clientReady = true;
                listeners.forEach((l) => l());
            },
            onError: (e) => {
                if (isDebugEnabled()) {
                    console.error("[useLspWorker] worker error", e);
                }
            },
            debug: isDebugEnabled(),
        });
    }
    return sharedClient;
}

/**
 * Start the shared worker and wait for it to be ready.
 * Safe to call multiple times - subsequent calls return the same promise.
 */
export function startSharedWorker(): Promise<void> {
    if (!initPromise) {
        const client = getSharedWorkerClient();
        initPromise = client.start();
    }
    return initPromise;
}

/**
 * Subscribe to LSP notifications from the worker.
 */
export function subscribeToNotifications(
    method: string,
    handler: LspNotificationHandler
): () => void {
    const sub: NotificationSubscriber = { method, handler };
    notificationSubscribers.add(sub);
    return () => {
        notificationSubscribers.delete(sub);
    };
}

// External store for React's useSyncExternalStore
function subscribe(callback: () => void): () => void {
    listeners.add(callback);
    return () => listeners.delete(callback);
}

function getSnapshot(): boolean {
    return clientReady;
}

export type UseLspWorkerResult = {
    status: "pending" | "ready" | "error";
    client: LspWorkerClient | null;
    error: Error | null;
};

/**
 * React hook that provides access to the shared LSP worker client.
 *
 * Usage:
 * ```tsx
 * const { status, client } = useLspWorker();
 *
 * if (status === "pending") return <div>Loading...</div>;
 *
 * // Use client.compileFiles(), client.match(), client.request(), etc.
 * ```
 */
export function useLspWorker(): UseLspWorkerResult {
    const [error, setError] = useState<Error | null>(null);
    const isReady = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
    useEffect(() => {
        startSharedWorker().catch(setError);
    }, []);
    if (error) {
        return { status: "error", client: null, error };
    }
    if (!isReady) {
        return { status: "pending", client: null, error: null };
    }
    return { status: "ready", client: getSharedWorkerClient(), error: null };
}

