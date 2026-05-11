import { useEffect, useRef, useState } from "react";
import type { EditorProps, Monaco } from "@monaco-editor/react";
import { Editor } from "@monaco-editor/react";
import { registerPegLanguage, PEG_THEME_ID, PEG_LIGHT_THEME_ID } from "../monaco/peg";
import TreeView from "../components/TreeView";
import TraceExplorer from "../components/TraceExplorer";
import SplitView from "../components/SplitView";
import { useLiveEditor, type LiveEditorSettings } from "./useLiveEditor";
import {
    LiveEditorRoot,
    LiveEditorBody,
    LiveEditorStatusBar,
    LiveEditorStatusDot,
    LiveEditorSide,
    LiveEditorPanelLabel,
    LiveEditorLoading,
    LiveEditorOutputPane,
    LiveEditorTabs,
    LiveEditorTab,
} from "./LiveEditor.styles";
import { ErrorDisplay } from "../Playground.styles";

const EDITOR_OPTIONS = {
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    lineNumbers: "off",
    folding: false,
    renderLineHighlight: "none",
    overviewRulerLanes: 0,
    fontSize: 13,
} satisfies EditorProps["options"];

export interface LiveEditorProps {
    grammar?: string;
    input?: string;
    height?: string;
    defaultView?: "tree" | "trace";
    showOutput?: boolean;
    settings?: LiveEditorSettings;
}

export default function LiveEditor({
    grammar: initialGrammar = "",
    input: initialInput = "",
    height = "380px",
    defaultView = "tree",
    showOutput = true,
    settings,
}: LiveEditorProps) {
    const [grammar, setGrammar] = useState(initialGrammar);
    const [input, setInput] = useState(initialInput);
    const [outputView, setOutputView] = useState<"tree" | "trace">(defaultView);

    const { result, outputError, hoverRange, setHoverRange, workerStatus } =
        useLiveEditor({ grammar, input, settings });

    const monacoRef = useRef<Monaco | null>(null);
    const inputEditorRef = useRef<any | null>(null);
    const inputDecorationsRef = useRef<string[]>([]);

    const [monacoTheme, setMonacoTheme] = useState<string>(PEG_LIGHT_THEME_ID);

    useEffect(() => {
        setMonacoTheme(
            document.documentElement.getAttribute("data-theme") === "dark"
                ? PEG_THEME_ID
                : PEG_LIGHT_THEME_ID,
        );
    }, []);

    useEffect(() => {
        const handler = (e: Event) => {
            const mode = (e as CustomEvent<string>).detail;
            const next = mode === "dark" ? PEG_THEME_ID : PEG_LIGHT_THEME_ID;
            setMonacoTheme(next);
            monacoRef.current?.editor.setTheme(next);
        };
        window.addEventListener("theme-change", handler);
        return () => window.removeEventListener("theme-change", handler);
    }, []);

    // Highlight the hovered range in the input editor via decorations
    useEffect(() => {
        const editor = inputEditorRef.current;
        const monaco = monacoRef.current;
        if (!editor || !monaco) return;
        const model = editor.getModel?.();
        if (!model) return;

        if (!hoverRange || hoverRange.end.utf16Cursor <= hoverRange.start.utf16Cursor) {
            inputDecorationsRef.current = editor.deltaDecorations(inputDecorationsRef.current, []);
            return;
        }

        const startPos = model.getPositionAt(hoverRange.start.utf16Cursor);
        const endPos = model.getPositionAt(hoverRange.end.utf16Cursor);
        const range = new monaco.Range(
            startPos.lineNumber, startPos.column,
            endPos.lineNumber, endPos.column,
        );
        inputDecorationsRef.current = editor.deltaDecorations(inputDecorationsRef.current, [{
            range,
            options: {
                inlineClassName: "ll-hover-highlight",
                stickiness: monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
            },
        }]);
    }, [hoverRange]);

    const beforeMount = (monaco: Monaco) => {
        monacoRef.current = monaco;
        registerPegLanguage(monaco);
    };

    const grammarEditor = (
        <LiveEditorSide style={{ flex: 1, minWidth: 0 }}>
            <LiveEditorPanelLabel>Grammar</LiveEditorPanelLabel>
            <div style={{ flex: 1, minHeight: 0 }}>
                <Editor
                    theme={monacoTheme}
                    language="peg"
                    height="100%"
                    width="100%"
                    value={grammar}
                    options={EDITOR_OPTIONS}
                    beforeMount={beforeMount}
                    onChange={(v) => setGrammar(v ?? "")}
                />
            </div>
        </LiveEditorSide>
    );

    const inputEditor = (
        <LiveEditorSide style={{ flex: 1, minWidth: 0 }}>
            <LiveEditorPanelLabel>Input</LiveEditorPanelLabel>
            <div style={{ flex: 1, minHeight: 0 }}>
                <Editor
                    theme={monacoTheme}
                    language="plaintext"
                    height="100%"
                    width="100%"
                    value={input}
                    options={EDITOR_OPTIONS}
                    onMount={(editor) => { inputEditorRef.current = editor; }}
                    onChange={(v) => setInput(v ?? "")}
                />
            </div>
        </LiveEditorSide>
    );

    const outputPanel = showOutput ? (
        <LiveEditorSide style={{ minWidth: "180px" }}>
            <LiveEditorTabs>
                <LiveEditorTab
                    type="button"
                    active={outputView === "tree"}
                    onClick={() => setOutputView("tree")}
                >
                    Tree
                </LiveEditorTab>
                <LiveEditorTab
                    type="button"
                    active={outputView === "trace"}
                    onClick={() => {
                        setOutputView("trace");
                        setHoverRange(null);
                    }}
                >
                    Trace
                </LiveEditorTab>
            </LiveEditorTabs>
            <LiveEditorOutputPane>
                {result ? (
                    outputView === "tree" ? (
                        <TreeView tree={result} onHoverRange={setHoverRange} />
                    ) : (
                        <TraceExplorer tree={result} onHoverRange={setHoverRange} />
                    )
                ) : outputError ? (
                    <ErrorDisplay style={{ fontSize: "0.8rem" }}>
                        {outputError}
                    </ErrorDisplay>
                ) : null}
            </LiveEditorOutputPane>
        </LiveEditorSide>
    ) : null;

    if (workerStatus === "pending") {
        return (
            <LiveEditorRoot style={{ height }}>
                <LiveEditorLoading>Initializing WASM…</LiveEditorLoading>
            </LiveEditorRoot>
        );
    }

    if (workerStatus === "error") {
        return (
            <LiveEditorRoot style={{ height }}>
                <LiveEditorLoading style={{ color: "rgba(255, 123, 123, 0.9)" }}>
                    Failed to load WASM runtime
                </LiveEditorLoading>
            </LiveEditorRoot>
        );
    }

    return (
        <LiveEditorRoot style={{ height }}>
            <LiveEditorBody>
                {showOutput ? (
                    <SplitView
                        initialRatio={0.6}
                        left={
                            <SplitView
                                initialRatio={0.55}
                                left={grammarEditor}
                                right={inputEditor}
                            />
                        }
                        right={outputPanel}
                    />
                ) : (
                    <SplitView
                        initialRatio={0.55}
                        top={grammarEditor}
                        bottom={inputEditor}
                    />
                )}
            </LiveEditorBody>
            <LiveEditorStatusBar>
                <LiveEditorStatusDot error={!!outputError} />
                {outputError ? "Error" : "Live"}
            </LiveEditorStatusBar>
        </LiveEditorRoot>
    );
}
