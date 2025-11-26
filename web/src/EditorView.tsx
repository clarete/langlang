import { Editor, type EditorProps } from "@monaco-editor/react";

import "./EditorView.css";
import { useCallback, useState } from "react";

interface EditorViewProps {
	grammar: string;
	input: string;
	onCompileRequest: (grammar: string, input: string) => void;
	children?: React.ReactNode;
}

const EDITOR_OPTIONS = {
	minimap: {
		enabled: false,
	},
	scrollBeyondLastLine: false,
} satisfies EditorProps["options"];

function EditorView({
	grammar,
	input,
	onCompileRequest,
	children,
}: EditorViewProps) {
	const [grammarText, setGrammarText] = useState(grammar);
	const [inputText, setInputText] = useState(input);

	const handleCompileRequest = useCallback(() => {
		onCompileRequest(grammarText, inputText);
	}, [grammarText, inputText, onCompileRequest]);

	return (
		<div className="root-container">
			<div className="editors">
				<div className="editor-container">
					<Editor
						theme="vs-dark"
						language="text"
						height="unset"
						width="100%"
						options={EDITOR_OPTIONS}
						value={grammar}
						onChange={(value) => {
							console.log(JSON.stringify(value, null, 2));
							setGrammarText(value ?? "");
						}}
					/>
				</div>
				<div style={{ gridColumn: "span 2" }}>
					<button
						type="button"
						id="compileAndMatch"
						onClick={handleCompileRequest}
						className="send-button"
					>
						Compile {"→"}
					</button>
				</div>
				<div className="editor-container">
					<Editor
						theme="vs-dark"
						language="text"
						height="unset"
						width="100%"
						options={EDITOR_OPTIONS}
						value={input}
						onChange={(value) => {
							setInputText(value ?? "");
						}}
					/>
				</div>
			</div>
			<div className="result-container">{children}</div>
		</div>
	);
}

export default EditorView;
