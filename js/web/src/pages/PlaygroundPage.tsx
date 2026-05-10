"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import { useLspWorker } from "../worker/useLspWorker";
import type { Value, Span } from "@langlang/wasm";
import type { EditorProps, Monaco } from "@monaco-editor/react";
import MatcherSettingsPanel from "../components/MatcherSettingsPanel";
import TraceExplorer from "../components/TraceExplorer";
import SplitView from "../components/SplitView";
import TreeView from "../components/TreeView";
import { registerPegLanguage, PEG_THEME_ID, PEG_LIGHT_THEME_ID } from "../monaco/peg";
import {
    ensureProjectModels,
    fromDocUri,
    startLanglangLsp,
} from "../monaco/lsp";
import WorkspaceSidebar from "../components/WorkspaceSidebar";
import EditorPanel from "../components/EditorPanel";
import ProjectPanel from "../components/ProjectPanel";
import { useWorkspacePlayground } from "../workspace/useWorkspacePlayground";
import { useDebouncedEffect } from "../live/useDebouncedEffect";

import {
    BarHeader,
    BarRoot,
    SettingsTab,
    Status,
    OutputPanelBody,
    OutputViewContainerWrapper,
    OutputView,
    ErrorDisplay,
} from "../Playground.styles";
import {
    PanelBody,
    PanelContainer,
    PanelHeader,
} from "../components/Panel.styles";
import { Tabs, Tab } from "../components/Tabs.styles";

function formatError(error: unknown): string {
    if (error instanceof Error) {
        return error.message;
    }
    return String(error);
}

const COMPILATION_DEBOUNCE_MS = 200;
const MATCHING_DEBOUNCE_MS = 50;

const EDITOR_OPTIONS = {
    minimap: {
        enabled: false,
    },
    scrollBeyondLastLine: false,
} satisfies EditorProps["options"];

function PlaygroundPage() {
    const [result, setResult] = useState<Value | null>(null);
    const [outputView, setOutputView] = useState<"tree" | "trace">("tree");
    const { status, client, error } = useLspWorker();

    // Store the matcher ID instead of the Matcher object
    const matcherIdRef = useRef<number | null>(null);
    const [outputError, setOutputError] = useState<string | null>(null);
    const [matcherVersion, setMatcherVersion] = useState(0);
    const [hoverRange, setHoverRange] = useState<Span | null>(null);
    const [isCompiling, setIsCompiling] = useState(false);
    const compileSeqRef = useRef(0);

    const [showSettings, setShowSettings] = useState(false);
    const [settings, setSettings] = useState(() => ({
        captureSpaces: false,
        handleSpaces: true,
        enableInline: true,
        showFails: true,
    }));

    const inputEditorRef = useRef<any | null>(null);
    const monacoRef = useRef<Monaco | null>(null);
    const inputDecorationsRef = useRef<string[]>([]);

    const [monacoTheme, setMonacoTheme] = useState<string>(() =>
        document.documentElement.getAttribute("data-theme") === "dark"
            ? PEG_THEME_ID
            : PEG_LIGHT_THEME_ID,
    );
    useEffect(() => {
        const handler = (e: Event) => {
            const mode = (e as CustomEvent<string>).detail;
            setMonacoTheme(mode === "dark" ? PEG_THEME_ID : PEG_LIGHT_THEME_ID);
        };
        window.addEventListener("theme-change", handler);
        return () => window.removeEventListener("theme-change", handler);
    }, []);

    const {
        workspace,
        selection,
        setSelection,
        project,
        activeGrammarPath,
        setActiveGrammarPath,
        activeGrammarContent,
        setActiveGrammarContent,
        inputs,
        activeInputPath,
        setActiveInputPath,
        activeInputContent,
        setActiveInputContent,
        createDir,
        createExample,
        rename,
        delete: deleteWsNode,
        selectGrammarFile,
    } = useWorkspacePlayground();

    const grammarEditorRef = useRef<any | null>(null);
    const grammarMonacoRef = useRef<Monaco | null>(null);

    const projectRef = useRef(project);
    useEffect(() => {
        projectRef.current = project;
    }, [project]);

    const registerMonacoLanguages = useCallback<
        NonNullable<EditorProps["beforeMount"]>
    >(
        (monaco) => {
            grammarMonacoRef.current = monaco;

            registerPegLanguage(monaco);

            // Ensure all project grammar files are actual Monaco
            // models so the Go LSP can resolve transitive
            // imports/defs without JS-side indexing.
            ensureProjectModels(monaco, projectRef.current.files);
            startLanglangLsp(monaco, {
                onNavigateToDefinition: (_from, toUri) => {
                    const path = fromDocUri(toUri);
                    if (path) setActiveGrammarPath(path);
                },
            });
        },
        [setActiveGrammarPath],
    );

    // Keep Monaco models in sync when switching examples/projects after mount.
    useEffect(() => {
        const monaco = grammarMonacoRef.current;
        if (!monaco) return;
        ensureProjectModels(monaco, project.files);
    }, [project]);

    const onGrammarEditorMount = useCallback<
        NonNullable<EditorProps["onMount"]>
    >(
        (editor, monaco) => {
            grammarEditorRef.current = editor;

            // Enable cross-file navigation in Monaco standalone.
            //
            // Monaco's default behavior is to do nothing when the
            // target URI belongs to a different model.  This hook is
            // invoked on actual navigation (click/F12), not on
            // Cmd-hover linkification.
            (monaco as any).editor.registerEditorOpener({
                openCodeEditor: (
                    source: any,
                    resource: any,
                    selectionOrPosition?: any,
                ) => {
                    const uriStr = String(resource);
                    const nextPath = fromDocUri(uriStr);
                    if (!nextPath) return false;

                    // Ensure the tab state follows navigation.
                    if (nextPath !== activeGrammarPath)
                        setActiveGrammarPath(nextPath);

                    // If the model exists, swap the editor to it immediately.
                    const model = (monaco as any).editor.getModel(resource);
                    if (model) {
                        source.setModel?.(model);
                        if (selectionOrPosition) {
                            if (
                                typeof selectionOrPosition.startLineNumber ===
                                "number"
                            ) {
                                source.setSelection?.(selectionOrPosition);
                                source.revealRangeInCenter?.(
                                    selectionOrPosition,
                                );
                            } else if (
                                typeof selectionOrPosition.lineNumber ===
                                "number"
                            ) {
                                source.setPosition?.(selectionOrPosition);
                                source.revealPositionInCenter?.(
                                    selectionOrPosition,
                                );
                            }
                        }
                        source.focus?.();
                    }
                    return true;
                },
            });

            // When Monaco navigates across models
            // (e.g. go-to-definition into another file), sync the
            // React state so the active tab/file updates too.
            editor.onDidChangeModel?.(() => {
                const model = editor.getModel?.();
                const uri = model?.uri;
                const uriStr = uri ? String(uri) : "";
                const nextPath = uri ? fromDocUri(uriStr) : null;
                if (!nextPath) return;
                if (nextPath && nextPath !== activeGrammarPath) {
                    setActiveGrammarPath(nextPath);
                }
            });
        },
        [activeGrammarPath, setActiveGrammarPath],
    );

    // Debounced compile step (grammar -> matcher)
    useDebouncedEffect(
        () => {
            if (!client) return;
            compileSeqRef.current += 1;
            const seq = compileSeqRef.current;
            setIsCompiling(true);
            setOutputError(null);

            // Free the previous matcher
            if (matcherIdRef.current !== null) {
                client.freeMatcher(matcherIdRef.current);
                matcherIdRef.current = null;
            }

            const matcherCfg: Record<string, unknown> = {
                "grammar.capture_spaces": settings.captureSpaces,
                "grammar.handle_spaces": settings.handleSpaces,
                "compiler.inline.enabled": settings.enableInline,
                "vm.show_fails": settings.showFails,
            };

            // Compile in the worker
            client
                .compileFiles(project.entry, project.files, matcherCfg)
                .then(({ matcherId }) => {
                    if (compileSeqRef.current !== seq) {
                        // Stale result, free the matcher
                        client.freeMatcher(matcherId);
                        return;
                    }
                    matcherIdRef.current = matcherId;
                    setOutputError(null);
                    setMatcherVersion((v) => v + 1);
                })
                .catch((e) => {
                    if (compileSeqRef.current !== seq) return;
                    const msg = formatError(e);
                    console.error(e);
                    setResult(null);
                    setOutputError(msg);
                })
                .finally(() => {
                    if (compileSeqRef.current === seq) {
                        setIsCompiling(false);
                    }
                });
        },
        [client, project, settings],
        COMPILATION_DEBOUNCE_MS,
    );

    // Debounced match step (matcher + input -> result tree)
    useDebouncedEffect(
        () => {
            if (isCompiling) {
                setOutputError(null);
                return;
            }
            if (!client) return;
            const matcherId = matcherIdRef.current;
            if (matcherId === null) return;

            client
                .match(matcherId, activeInputContent)
                .then(({ value }) => {
                    setResult(value as Value);
                    setOutputError(null);
                })
                .catch((e) => {
                    const msg = formatError(e);
                    console.error(e);
                    setResult(null);
                    setOutputError(msg);
                });
        },
        [client, activeInputContent, matcherVersion, isCompiling],
        MATCHING_DEBOUNCE_MS,
    );

    // Hover highlight (tree node range -> input editor decoration)
    useEffect(() => {
        const editor = inputEditorRef.current;
        const monaco = monacoRef.current;
        if (!editor || !monaco) return;
        const model = editor.getModel?.();
        if (!model) return;

        // nothing's selected, so we clear whatever decoration is in
        // the selection
        if (
            !hoverRange ||
            hoverRange.end.utf16Cursor <= hoverRange.start.utf16Cursor
        ) {
            inputDecorationsRef.current = editor.deltaDecorations(
                inputDecorationsRef.current,
                [],
            );
            return;
        }

        const startPos = model.getPositionAt(hoverRange.start.utf16Cursor);
        const endPos = model.getPositionAt(hoverRange.end.utf16Cursor);
        const range = new monaco.Range(
            startPos.lineNumber,
            startPos.column,
            endPos.lineNumber,
            endPos.column,
        );

        inputDecorationsRef.current = editor.deltaDecorations(
            inputDecorationsRef.current,
            [
                {
                    range,
                    options: {
                        inlineClassName: "ll-hover-highlight",
                        stickiness:
                            monaco.editor.TrackedRangeStickiness
                                .NeverGrowsWhenTypingAtEdges,
                    },
                },
            ],
        );
    }, [hoverRange]);

    // Cleanup on unmount
    useEffect(() => {
        return () => {
            if (matcherIdRef.current !== null && client) {
                client.freeMatcher(matcherIdRef.current);
                matcherIdRef.current = null;
            }
        };
    }, [client]);

    const pageStyle: React.CSSProperties = {
        display: "flex",
        flexDirection: "column",
        height: "100%",
    };

    if (status === "pending") {
        return <div style={pageStyle} />;
    }

    if (status === "error") {
        return (
            <div style={{ ...pageStyle, padding: "1rem", fontFamily: "monospace", fontSize: "0.85rem" }}>
                Error: {error?.message}
            </div>
        );
    }

    if (status === "ready") {
        return (
            <div style={pageStyle}>
                <div style={{ display: "flex", flex: 1, minHeight: 0 }}>
                <SplitView
                    initialRatio={0.65}
                    left={
                        <SplitView
                            initialRatio={0.25}
                            left={
                                workspace ? (
                                    <WorkspaceSidebar
                                        workspace={workspace}
                                        selection={selection}
                                        onSelect={setSelection}
                                        onSelectGrammarFile={selectGrammarFile}
                                        onCreateDir={createDir}
                                        onCreateExample={createExample}
                                        onRenameNode={rename}
                                        onDeleteNode={deleteWsNode}
                                    />
                                ) : (
                                    <div />
                                )
                            }
                            right={
                                <SplitView
                                    initialRatio={0.6}
                                    top={
                                        <ProjectPanel
                                            project={project}
                                            activePath={activeGrammarPath}
                                            onActivePathChange={
                                                setActiveGrammarPath
                                            }
                                            value={activeGrammarContent}
                                            onChange={setActiveGrammarContent}
                                            options={EDITOR_OPTIONS}
                                            beforeMount={
                                                registerMonacoLanguages
                                            }
                                            onMount={onGrammarEditorMount}
                                            theme={monacoTheme}
                                        />
                                    }
                                    bottom={
                                        <EditorPanel
                                            title="Input"
                                            language="text"
                                            value={activeInputContent}
                                            options={EDITOR_OPTIONS}
                                            theme={monacoTheme}
                                            onMount={(editor, monaco) => {
                                                inputEditorRef.current = editor;
                                                monacoRef.current = monaco;
                                            }}
                                            onChange={setActiveInputContent}
                                            headerRight={
                                                inputs.length > 1 ? (
                                                    <select
                                                        value={activeInputPath}
                                                        onChange={(e) =>
                                                            setActiveInputPath(
                                                                e.target.value,
                                                            )
                                                        }
                                                    >
                                                        {inputs.map((f) => (
                                                            <option
                                                                key={f.path}
                                                                value={f.path}
                                                            >
                                                                {f.path}
                                                            </option>
                                                        ))}
                                                    </select>
                                                ) : null
                                            }
                                        />
                                    }
                                />
                            }
                        />
                    }
                    right={
                        <PanelContainer>
                            <PanelHeader>
                                <Tabs>
                                    <Tab
                                        type="button"
                                        active={outputView === "tree"}
                                        onClick={() => setOutputView("tree")}
                                    >
                                        Tree
                                    </Tab>
                                    <Tab
                                        type="button"
                                        active={outputView === "trace"}
                                        onClick={() => {
                                            setOutputView("trace");
                                            setHoverRange(null);
                                        }}
                                    >
                                        Trace
                                    </Tab>
                                </Tabs>
                            </PanelHeader>
                            <PanelBody
                                style={{
                                    display: "flex",
                                    flexDirection: "column",
                                    minHeight: 0,
                                }}
                            >
                                <OutputPanelBody style={{ minHeight: 0 }}>
                                    <OutputView>
                                        <OutputViewContainerWrapper>
                                            {result ? (
                                                outputView === "tree" ? (
                                                    <TreeView
                                                        tree={result}
                                                        onHoverRange={
                                                            setHoverRange
                                                        }
                                                    />
                                                ) : (
                                                    <TraceExplorer
                                                        tree={result}
                                                        onHoverRange={
                                                            setHoverRange
                                                        }
                                                    />
                                                )
                                            ) : outputError ? (
                                                <ErrorDisplay>
                                                    {outputError}
                                                </ErrorDisplay>
                                            ) : (
                                                ""
                                            )}
                                        </OutputViewContainerWrapper>
                                    </OutputView>
                                </OutputPanelBody>
                            </PanelBody>
                        </PanelContainer>
                    }
                />
                </div>
                <BarRoot>
                    <BarHeader>
                        <SettingsTab
                            type="button"
                            active={showSettings}
                            aria-expanded={showSettings}
                            onClick={() => setShowSettings((v) => !v)}
                        >
                            Settings
                        </SettingsTab>
                        <Status
                            title={outputError ?? "Live preview"}
                            style={{
                                color: outputError
                                    ? "rgba(255, 123, 123, 0.9)"
                                    : "rgba(123, 255, 180, 0.9)",
                            }}
                        >
                            {outputError ? "Error" : "Live"}
                        </Status>
                    </BarHeader>
                    {showSettings ? (
                        <MatcherSettingsPanel
                            value={settings}
                            onChange={setSettings}
                        />
                    ) : null}
                </BarRoot>
            </div>
        );
    }
}

export default PlaygroundPage;
