"use client";

import { useEffect, useRef, useState } from "react";
import { useWasmTest, type Matcher, type Value } from "@langlang/react";
import { Editor, type EditorProps } from "@monaco-editor/react";
import TraceExplorer from "./components/TraceExplorer";
import SplitView from "./components/SplitView";
import TreeView from "./components/TreeView";
import File from "./components/File";
import { registerPegLanguage } from "./monaco/peg";

import {
    OutputPanelBody,
    OutputTab,
    OutputTabs,
    OutputViewContainerWrapper,
    PanelBody,
    PanelContainer,
    PanelHeader,
    TopBar,
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
                matcherRef.current = langlang.matcherFromString(grammarText);
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
    }, [langlang, grammarText]);

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

    const handleGrammarChange = (value: string) => {
        setGrammarText(fixtures[value as keyof typeof fixtures].grammar);
        setInputText(fixtures[value as keyof typeof fixtures].input);
    };

    if (status === "pending") {
        return <div>Loading...</div>;
    }

    if (status === "error") {
        return <div>Error: {error?.message}</div>;
    }

    if (status === "success") {
        return (
            <>
                <TopBar>
                    <select
                        defaultValue="protoCirc"
                        onChange={(e) =>
                            handleGrammarChange(
                                e.target.value as keyof typeof fixtures,
                            )
                        }
                    >
                        <option value="demo">Demo</option>
                        <option value="json">JSON</option>
                        <option value="jsonStripped">JSON Stripped</option>
                        <option value="csv">CSV</option>
                        <option value="langlang">LangLang</option>
                        <option value="xmlUnstable">XML Unstable</option>
                        <option value="protoCirc">Proto Circ</option>
                    </select>
                    <div
                        title={outputError ?? "Live preview"}
                        style={{
                            fontFamily:
                                "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
                            fontSize: "0.8rem",
                            color: outputError
                                ? "rgba(255, 123, 123, 0.9)"
                                : "rgba(123, 255, 180, 0.9)",
                            marginLeft: "auto",
                        }}
                    >
                        {outputError ? "Error" : "Live"}
                    </div>
                </TopBar>

                <SplitView
                    left={
                        <SplitView
                            top={
                                <PanelContainer>
                                    <PanelHeader>Grammar (PEG)</PanelHeader>
                                    <PanelBody>
                                        <Editor
                                            theme="vs-dark"
                                            beforeMount={
                                                registerMonacoLanguages
                                            }
                                            language="peg"
                                            height="100%"
                                            width="100%"
                                            options={EDITOR_OPTIONS}
                                            value={grammarText}
                                            onChange={(value) => {
                                                setGrammarText(value ?? "");
                                            }}
                                        />
                                    </PanelBody>
                                </PanelContainer>
                            }
                            bottom={
                                <PanelContainer>
                                    <PanelHeader>Input</PanelHeader>
                                    <PanelBody>
                                        <Editor
                                            theme="vs-dark"
                                            language="text"
                                            height="100%"
                                            width="100%"
                                            options={EDITOR_OPTIONS}
                                            value={inputText}
                                            onChange={(value) => {
                                                setInputText(value ?? "");
                                            }}
                                        />
                                    </PanelBody>
                                </PanelContainer>
                            }
                        />
                    }
                    right={
                        <PanelContainer>
                            <PanelHeader>Output</PanelHeader>
                            <PanelBody>
                                <OutputPanelBody>
                                    <OutputViewContainerWrapper>
                                        {result ? (
                                            outputView === "tree" ? (
                                                <TreeView tree={result} />
                                            ) : (
                                                <TraceExplorer tree={result} />
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
                                            onClick={() =>
                                                setOutputView("trace")
                                            }
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
