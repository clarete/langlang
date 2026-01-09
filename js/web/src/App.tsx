"use client";

import { useEffect, useRef, useState } from "react";
import { useWasmTest } from "@langlang/react";
import type { Config, Matcher, Value, Span } from "@langlang/react";
import { Editor, type EditorProps, type Monaco } from "@monaco-editor/react";
import MatcherSettingsPanel from "./components/MatcherSettingsPanel";
import TraceExplorer from "./components/TraceExplorer";
import SplitView from "./components/SplitView";
import TreeView from "./components/TreeView";
import File from "./components/File";
import { registerPegLanguage } from "./monaco/peg";

import {
    BarHeader,
    BarRoot,
    BarSpacer,
    BarTitle,
    SettingsTab,
    Status,
    OutputPanelBody,
    OutputTab,
    OutputTabs,
    OutputViewContainerWrapper,
    PanelBody,
    PanelContainer,
    PanelHeader,
} from "./App.styles";

const EDITOR_OPTIONS = {
    minimap: {
        enabled: false,
    },
    scrollBeyondLastLine: false,
} satisfies EditorProps["options"];

const registerMonacoLanguages: EditorProps["beforeMount"] = (monaco) => {
    registerPegLanguage(monaco);
};

function App() {
    const [result, setResult] = useState<Value | null>(null);
    const [outputView, setOutputView] = useState<"tree" | "trace">("tree");
    const [grammarText, setGrammarText] = useState<string>("");
    const [inputText, setInputText] = useState<string>("");
    const { status, data: langlang, error } = useWasmTest();

    const matcherRef = useRef<Matcher | null>(null);
    const [outputError, setOutputError] = useState<string | null>(null);
    const [matcherVersion, setMatcherVersion] = useState(0);
    const [hoverRange, setHoverRange] = useState<Span | null>(null);

    const [showSettings, setShowSettings] = useState(false);
    const [settings, setSettings] = useState(() => ({
        captureSpaces: false,
        handleSpaces: true,
        enableInline: true,
        showFails: true,
    }));

    const inputEditorRef = useRef<any>(null);
    const monacoRef = useRef<Monaco | null>(null);
    const inputDecorationsRef = useRef<string[]>([]);

    // Debounced compile step (grammar -> matcher)
    useEffect(() => {
        if (!langlang) return;
        const handle = window.setTimeout(() => {
            // dispose the previous matcher
            try {
                matcherRef.current?.dispose();
            } catch (_) {
                // ignore
            } finally {
                matcherRef.current = null;
            }
            try {
                const matcherCfg: Config = {
                    "grammar.capture_spaces": settings.captureSpaces,
                    "grammar.handle_spaces": settings.handleSpaces,
                    "compiler.inline.enabled": settings.enableInline,
                    "vm.show_fails": settings.showFails,
                };
                matcherRef.current = langlang.matcherFromString(
                    grammarText,
                    matcherCfg,
                );
                setOutputError(null);
                setMatcherVersion((v) => v + 1);
            } catch (e) {
                const msg = e instanceof Error ? e.message : String(e);
                console.error(e);
                setResult(null);
                setOutputError(msg);
            }
        }, 200);

        return () => window.clearTimeout(handle);
    }, [langlang, grammarText, settings]);

    // Debounced match step (matcher + input -> result tree)
    useEffect(() => {
        const m = matcherRef.current;
        if (!m) return;

        const handle = window.setTimeout(() => {
            try {
                const { value } = m.match(inputText);
                setResult(value);
                setOutputError(null);
            } catch (e) {
                const msg = e instanceof Error ? e.message : String(e);
                console.error(e);
                setResult(null);
                setOutputError(msg);
            }
        }, 50);

        return () => window.clearTimeout(handle);
    }, [inputText, matcherVersion]);

    // Hover highlight (tree node range -> input editor decoration)
    useEffect(() => {
        const editor = inputEditorRef.current;
        const monaco = monacoRef.current;
        if (!editor || !monaco) return;
        const model = editor.getModel?.();
        if (!model) return;

        // nothing's selected, so we clear whatever decoration is in the selection
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
            try {
                matcherRef.current?.dispose();
            } catch (_) {
                // ignore
            } finally {
                matcherRef.current = null;
            }
        };
    }, []);

    if (status === "pending") {
        return <div>Loading...</div>;
    }

    if (status === "error") {
        return <div>Error: {error?.message}</div>;
    }

    if (status === "success") {
        return (
            <>
                <BarRoot>
                    <BarHeader>
                        <BarTitle>langlang</BarTitle>
                        <BarSpacer />
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
                <SplitView
                    left={
                        <SplitView
                            top={
                                <PanelContainer>
                                    <PanelBody>
                                        <File
                                            title="Grammar (PEG)"
                                            selectUrlFromPair={({
                                                grammar: url,
                                                id,
                                            }) => ({
                                                type: "url",
                                                url,
                                                id,
                                            })}
                                            onContentChange={(content) =>
                                                setGrammarText(content ?? "")
                                            }
                                            accept={{
                                                "text/plain": [".peg"],
                                            }}
                                        >
                                            {(content, write) => (
                                                <Editor
                                                    theme="vs-dark"
                                                    beforeMount={
                                                        registerMonacoLanguages
                                                    }
                                                    language="peg"
                                                    height="100%"
                                                    width="100%"
                                                    options={EDITOR_OPTIONS}
                                                    value={content ?? undefined}
                                                    onChange={(value) => {
                                                        write(value ?? "");
                                                    }}
                                                />
                                            )}
                                        </File>
                                    </PanelBody>
                                </PanelContainer>
                            }
                            bottom={
                                <PanelContainer>
                                    <PanelBody>
                                        <File
                                            title="Input"
                                            selectUrlFromPair={(pair) => ({
                                                type: "url",
                                                url: pair.input,
                                                id: pair.id,
                                            })}
                                            onContentChange={(content) =>
                                                setInputText(content ?? "")
                                            }
                                        >
                                            {(content, write) => (
                                                <Editor
                                                    theme="vs-dark"
                                                    language="text"
                                                    height="100%"
                                                    width="100%"
                                                    options={EDITOR_OPTIONS}
                                                    value={content ?? undefined}
                                                    onMount={(
                                                        editor,
                                                        monaco,
                                                    ) => {
                                                        inputEditorRef.current =
                                                            editor;
                                                        monacoRef.current =
                                                            monaco;
                                                    }}
                                                    onChange={(value) =>
                                                        write(value ?? "")
                                                    }
                                                />
                                            )}
                                        </File>
                                    </PanelBody>
                                </PanelContainer>
                            }
                        />
                    }
                    right={
                        <PanelContainer>
                            <PanelHeader>Output</PanelHeader>
                            <PanelBody
                                style={{
                                    display: "flex",
                                    flexDirection: "column",
                                    minHeight: 0,
                                }}
                            >
                                <OutputPanelBody style={{ minHeight: 0 }}>
                                    <OutputViewContainerWrapper>
                                        {result ? (
                                            outputView === "tree" ? (
                                                <TreeView
                                                    tree={result}
                                                    onHoverRange={setHoverRange}
                                                />
                                            ) : (
                                                <TraceExplorer
                                                    tree={result}
                                                    onHoverRange={setHoverRange}
                                                />
                                            )
                                        ) : outputError ? (
                                            <div
                                                style={{
                                                    padding: "0.75rem",
                                                    color: "rgba(255, 123, 123, 0.9)",
                                                    fontFamily:
                                                        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
                                                    whiteSpace: "pre-wrap",
                                                }}
                                            >
                                                {outputError}
                                            </div>
                                        ) : (
                                            ""
                                        )}
                                    </OutputViewContainerWrapper>
                                    <OutputTabs>
                                        <OutputTab
                                            type="button"
                                            active={outputView === "tree"}
                                            onClick={() =>
                                                setOutputView("tree")
                                            }
                                        >
                                            Tree
                                        </OutputTab>
                                        <OutputTab
                                            type="button"
                                            active={outputView === "trace"}
                                            onClick={() => {
                                                setOutputView("trace");
                                                setHoverRange(null);
                                            }}
                                        >
                                            Trace
                                        </OutputTab>
                                    </OutputTabs>
                                </OutputPanelBody>
                            </PanelBody>
                        </PanelContainer>
                    }
                />
            </>
        );
    }
}

export default App;
