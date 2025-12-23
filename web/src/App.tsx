"use client";

import { useState } from "react";

// @ts-expect-error - wasmUrl is a file URL
import wasmUrl from "../langlang.wasm?url" with { type: "file" };
import { useWasmTest, type LangLangValue } from "@langlang/react";

import fixtures from "./fixtures";

const EDITOR_OPTIONS = {
	minimap: {
		enabled: false,
	},
	scrollBeyondLastLine: false,
} satisfies EditorProps["options"];

import {
	EditorContainer,
	ResponseArea,
	ResponseContainer,
	SendButton,
	TopBar,
	TreeViewContainerWrapper,
} from "./App.styles";

import TreeExplorer from "./components/TreeExplorer";
import SplitView from "./components/SplitView";
import { Editor, type EditorProps } from "@monaco-editor/react";
import { compileZig } from "./circ/compiler";

function App() {
	const [result, setResult] = useState<LangLangValue | null>(null);

	const [grammarText, setGrammarText] = useState<string>(fixtures.json.grammar);
	const [inputText, setInputText] = useState<string>(fixtures.json.input);

	if (!wasmUrl) {
		throw new Error("WASM URL is not set");
	}

	const { status, data, error } = useWasmTest(wasmUrl);

	const handleCompileJson = (grammar: string, input: string) => {
		if (!data) {
			return;
		}

		try {
			const result = data.compileJson(grammar, input);
			console.log(result);

			setResult(result);
		} catch (error) {
			console.error(error);
			setResult(null);
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
							handleGrammarChange(e.target.value as keyof typeof fixtures)
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
							onClick={() => handleCompileJson(grammarText, inputText)}
						>
							Compile {"→"}
						</SendButton>
					</div>
				</TopBar>

				<SplitView
					left={
						<SplitView
							top={
								<EditorContainer>
									<Editor
										theme="vs-dark"
										language="text"
										height="100%"
										width="100%"
										options={EDITOR_OPTIONS}
										value={grammarText}
										onChange={(value) => {
											setGrammarText(value ?? "");
										}}
									/>
								</EditorContainer>
							}
							bottom={
								<EditorContainer>
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
								</EditorContainer>
							}
						/>
					}
					right={
						result ? (
							<SplitView
								top={
									<TreeViewContainerWrapper>
										<TreeExplorer tree={result} />
									</TreeViewContainerWrapper>
								}
								bottom={
									<ResponseContainer>
										<ResponseArea value={compileZig(result)} rows={30} />
									</ResponseContainer>
								}
							/>
						) : null
					}
				/>
			</>
		);
	}
}

export default App;
