/**
 * Worker module exports.
 *
 * The main entry point is useLspWorker hook which provides access to
 * the shared WASM worker for LSP and grammar compilation/matching.
 */

export { createLspWorkerClient, type LspWorkerClient, type LspClientConfig, type LspNotificationHandler } from "./lsp.client";
export { useLspWorker, getSharedWorkerClient, startSharedWorker, subscribeToNotifications } from "./useLspWorker";
export type { WorkerRequest, WorkerResponse } from "./lsp.worker";

