import type { Monaco } from "@monaco-editor/react";
import type { LspWorkerClient } from "../worker/lsp.client";
import {
    getSharedWorkerClient,
    startSharedWorker,
    subscribeToNotifications,
} from "../worker/useLspWorker";

const DOC_SCHEME = "inmemory";
const DOC_AUTHORITY = "project";

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

function dbg(...args: any[]) {
    if (!isDebugEnabled()) return;
    // eslint-disable-next-line no-console
    console.log("[langlang:lsp]", ...args);
}

export function toDocUri(path: string): string {
    const clean = path.startsWith("/") ? path : `/${path}`;
    return `${DOC_SCHEME}://${DOC_AUTHORITY}${clean}`;
}

export function fromDocUri(uri: string): string | null {
    const prefix = `${DOC_SCHEME}://${DOC_AUTHORITY}/`;
    if (!uri.startsWith(prefix)) return null;
    return uri.slice(prefix.length);
}

export function ensureProjectModels(
    monaco: Monaco,
    files: Array<{ path: string; content: string }>,
) {
    for (const f of files) {
        const uri = monaco.Uri.parse(toDocUri(f.path));
        const existing = monaco.editor.getModel(uri);
        if (existing) {
            // Ensure correct language even if the model was created elsewhere
            // before we registered the PEG language.
            if (existing.getLanguageId() !== "peg") {
                (monaco as any).editor.setModelLanguage(existing, "peg");
            }
            // Keep model content in sync on project switches.
            if (existing.getValue() !== f.content) {
                existing.setValue(f.content);
            }
            continue;
        }
        monaco.editor.createModel(f.content, "peg", uri);
    }

    // If the LSP is already running, make sure these models are opened there too.
    syncOpenPegModels(monaco);
}

let started = false;
let client: LanglangLspClient | null = null;
let workerClient: LspWorkerClient | null = null;
const openedUris = new Set<string>();
const changeHookedUris = new Set<string>();

function syncOpenPegModels(monaco: Monaco) {
    if (!workerClient) return;
    const wc = workerClient;
    for (const model of monaco.editor.getModels()) {
        if (model.getLanguageId() !== "peg") continue;
        const uri = model.uri.toString();
        if (!openedUris.has(uri)) {
            openedUris.add(uri);
            wc.notify("textDocument/didOpen", {
                textDocument: {
                    uri,
                    languageId: model.getLanguageId(),
                    version: model.getVersionId(),
                    text: model.getValue(),
                },
            });
        }
    }
}

export type LanglangLspHooks = {
    onDiagnostics?: (uri: string, diagnostics: Array<any>) => void;
    onNavigateToDefinition?: (
        fromUri: string,
        toUri: string,
        toRange?: any,
    ) => void;
};

export type LanglangLspClient = {
    requestDefinition: (
        uri: string,
        line: number,
        character: number,
    ) => Promise<any>;
    /** Access to the underlying worker client for grammar compilation */
    workerClient: LspWorkerClient;
};

export function startLanglangLsp(
    monaco: Monaco,
    hooks?: LanglangLspHooks,
): LanglangLspClient {
    if (started && client) {
        syncOpenPegModels(monaco);
        return client;
    }
    started = true;

    dbg("start");

    const changeTimers = new Map<string, number>();

    function lspToMonacoRange(range: {
        start: { line: number; character: number };
        end: { line: number; character: number };
    }) {
        // Monaco is 1-based, LSP is 0-based
        return new monaco.Range(
            range.start.line + 1,
            range.start.character + 1,
            range.end.line + 1,
            range.end.character + 1,
        );
    }

    // LSP and Monaco use identical numbering for SymbolKind
    function lspSymbolKindToMonaco(_monaco: Monaco, kind: number) {
        return kind;
    }

    // LSP and Monaco use identical numbering for CompletionItemKind
    function lspCompletionKindToMonaco(_monaco: Monaco, kind: number) {
        return kind;
    }

    function getHoverMarkdown(contents: any): string {
        // LSP Hover.contents can be:
        // - MarkupContent { kind, value }
        // - string / MarkedString
        // - array of MarkedString
        if (!contents) return "";
        else if (typeof contents === "string") return contents;
        else if (typeof contents?.value === "string") return contents.value;
        else if (Array.isArray(contents)) {
            return contents
                .map((c) =>
                    typeof c === "string"
                        ? c
                        : typeof c?.value === "string"
                          ? c.value
                          : "",
                )
                .filter(Boolean)
                .join("\n\n");
        }
        return "";
    }

    function openModel(model: any) {
        const uri = model.uri.toString();
        openedUris.add(uri);
        workerClient!.notify("textDocument/didOpen", {
            textDocument: {
                uri,
                languageId: model.getLanguageId(),
                version: model.getVersionId(),
                text: model.getValue(),
            },
        });
    }

    function changeModel(model: any) {
        const uri = model.uri.toString();
        const prev = changeTimers.get(uri);
        if (typeof prev === "number") window.clearTimeout(prev);
        const handle = window.setTimeout(() => {
            workerClient!.notify("textDocument/didChange", {
                textDocument: {
                    uri,
                    version: model.getVersionId(),
                },
                contentChanges: [{ text: model.getValue() }],
            });
        }, 100);
        changeTimers.set(uri, handle);
    }

    // Get the shared worker client (reuse the same worker for LSP and compilation)
    workerClient = getSharedWorkerClient();

    // Subscribe to diagnostics notifications
    subscribeToNotifications("textDocument/publishDiagnostics", (method, params) => {
        dbg("notification", method, params);
        const { uri, diagnostics } = params as {
            uri: string;
            diagnostics: Array<any>;
        };
        const model = monaco.editor.getModel(monaco.Uri.parse(uri));
        if (!model) return;
        const markers = (diagnostics ?? []).map((d) => ({
            message: d.message ?? "diagnostic",
            severity:
                d.severity === 1
                    ? monaco.MarkerSeverity.Error
                    : d.severity === 2
                      ? monaco.MarkerSeverity.Warning
                      : monaco.MarkerSeverity.Info,
            startLineNumber: d.range.start.line + 1,
            startColumn: d.range.start.character + 1,
            endLineNumber: d.range.end.line + 1,
            endColumn: d.range.end.character + 1,
        }));
        monaco.editor.setModelMarkers(model, "langlang", markers);
        hooks?.onDiagnostics?.(uri, diagnostics ?? []);
    });

    const readyPromise = startSharedWorker().then(async () => {
        dbg("worker ready");

        // Initialize the LSP
        await workerClient!.request("initialize", {
            processId: null,
            rootUri: null,
            capabilities: {},
        });
        await workerClient!.notify("initialized", {});

        // Open all currently known PEG models
        for (const model of monaco.editor.getModels()) {
            if (model.getLanguageId() === "peg") {
                openModel(model);
            }
        }

        monaco.editor.onDidCreateModel((model: any) => {
            if (model.getLanguageId() === "peg") {
                openModel(model);
            }
        });

        // Set up change tracking
        for (const model of monaco.editor.getModels()) {
            if (model.getLanguageId() !== "peg") continue;
            const uri = model.uri.toString();
            if (changeHookedUris.has(uri)) continue;
            changeHookedUris.add(uri);
            model.onDidChangeContent(() => changeModel(model));
        }

        monaco.editor.onDidCreateModel((model: any) => {
            if (model.getLanguageId() !== "peg") return;
            const uri = model.uri.toString();
            if (changeHookedUris.has(uri)) return;
            changeHookedUris.add(uri);
            model.onDidChangeContent(() => changeModel(model));
        });

        // Register Monaco providers
        monaco.languages.registerDefinitionProvider("peg", {
            provideDefinition: async (model: any, position: any) => {
                const uri = model.uri.toString();
                dbg("definition:request", { uri, position });
                const res = await workerClient!.request("textDocument/definition", {
                    textDocument: { uri },
                    position: {
                        line: position.lineNumber - 1,
                        character: position.column - 1,
                    },
                });
                dbg("definition:result", res);
                if (!res) return null;
                const locs = Array.isArray(res) ? res : [res];
                return locs.map((loc: any) => ({
                    uri: monaco.Uri.parse(loc.uri),
                    range: lspToMonacoRange(loc.range),
                }));
            },
        });

        monaco.languages.registerHoverProvider("peg", {
            provideHover: async (model: any, position: any) => {
                const uri = model.uri.toString();
                dbg("hover:request", { uri, position });
                const res = await workerClient!.request("textDocument/hover", {
                    textDocument: { uri },
                    position: {
                        line: position.lineNumber - 1,
                        character: position.column - 1,
                    },
                });
                dbg("hover:result", res);
                if (!res || !res.contents) return null;
                const value = getHoverMarkdown(res.contents);
                if (!value) return null;
                const range = res.range ? lspToMonacoRange(res.range) : undefined;
                return { range, contents: [{ value }] };
            },
        });

        monaco.languages.registerDocumentSymbolProvider("peg", {
            provideDocumentSymbols: async (model: any) => {
                const uri = model.uri.toString();
                dbg("documentSymbol:request", { uri });
                const res = await workerClient!.request("textDocument/documentSymbol", {
                    textDocument: { uri },
                });
                dbg("documentSymbol:result", res);
                if (!res || !Array.isArray(res)) return [];
                return res.map((sym: any) => ({
                    name: sym.name,
                    detail: sym.detail ?? "",
                    kind: lspSymbolKindToMonaco(monaco, sym.kind),
                    range: lspToMonacoRange(sym.range),
                    selectionRange: lspToMonacoRange(sym.selectionRange ?? sym.range),
                    children: sym.children?.map((child: any) => ({
                        name: child.name,
                        detail: child.detail ?? "",
                        kind: lspSymbolKindToMonaco(monaco, child.kind),
                        range: lspToMonacoRange(child.range),
                        selectionRange: lspToMonacoRange(child.selectionRange ?? child.range),
                    })),
                }));
            },
        });

        monaco.languages.registerCompletionItemProvider("peg", {
            triggerCharacters: ["."],
            provideCompletionItems: async (model: any, position: any) => {
                const uri = model.uri.toString();
                dbg("completion:request", { uri, position });
                const res = await workerClient!.request("textDocument/completion", {
                    textDocument: { uri },
                    position: {
                        line: position.lineNumber - 1,
                        character: position.column - 1,
                    },
                });
                dbg("completion:result", res);
                if (!res || !Array.isArray(res)) return { suggestions: [] };
                const word = model.getWordUntilPosition(position);
                const range = new monaco.Range(
                    position.lineNumber,
                    word.startColumn,
                    position.lineNumber,
                    word.endColumn,
                );
                const suggestions = res.map((item: any) => ({
                    label: item.label,
                    kind: lspCompletionKindToMonaco(monaco, item.kind),
                    detail: item.detail ?? "",
                    documentation: item.documentation?.value ?? "",
                    insertText: item.insertText ?? item.label,
                    insertTextRules: item.insertText?.includes("$")
                        ? monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet
                        : undefined,
                    range,
                }));
                return { suggestions };
            },
        });

        dbg("providers:registered");
    });

    client = {
        requestDefinition: async (uri: string, line: number, character: number) => {
            await readyPromise;
            return workerClient!.request("textDocument/definition", {
                textDocument: { uri },
                position: { line, character },
            });
        },
        workerClient: workerClient!,
    };

    return client;
}
