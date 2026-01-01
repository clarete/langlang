"use client";

import { useState } from "react";
import { useWasmTest, type Value } from "@langlang/react";
import { Editor, type EditorProps } from "@monaco-editor/react";
import TreeExplorer from "./components/TreeExplorer";
import SplitView from "./components/SplitView";

import fixtures from "./fixtures";
import { registerPegLanguage } from "./monaco/peg";

import {
    PanelBody,
    PanelContainer,
    PanelHeader,
    SendButton,
    TopBar,
    TreeViewContainerWrapper,
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
    const [grammarText, setGrammarText] = useState<string>(
        fixtures.protoCirc.grammar,
    );
    const [inputText, setInputText] = useState<string>(
        fixtures.protoCirc.input,
    );
    const { status, data: langlang, error } = useWasmTest();

    const handleCompileJson = (grammar: string, input: string) => {
        if (!langlang) {
            return;
        }
        const matcher = langlang.matcherFromString(grammar);
        try {
            const { value } = matcher.match(input);
            setResult(value);
        } catch (error) {
            console.error(error);
            setResult(null);
        } finally {
            matcher.dispose();
        }
    };

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
                    <div style={{ gridColumn: "span 2" }}>
                        <SendButton
                            type="button"
                            id="compileAndMatch"
                            onClick={() =>
                                handleCompileJson(grammarText, inputText)
                            }
                        >
                            Compile {"→"}
                        </SendButton>
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
                                            beforeMount={registerMonacoLanguages}
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
                        result ? (
                            <PanelContainer>
                                <PanelHeader>Output</PanelHeader>
                                <PanelBody>
                                    <TreeViewContainerWrapper>
                                        <TreeExplorer tree={result} />
                                    </TreeViewContainerWrapper>
                                </PanelBody>
                            </PanelContainer>
                        ) : null
                    }
                />
            </>
        );
    }
}

export default App;
