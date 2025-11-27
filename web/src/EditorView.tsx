import { Editor, type EditorProps } from "@monaco-editor/react";

import { useCallback, useState } from "react";
import { SendButton } from "./App.styles";
import {
	EditorContainer,
	Editors,
	ResultContainer,
	RootContainer,
} from "./EditorView.styles";

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
		<RootContainer>
			<Editors>
				<EditorContainer>
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
				</EditorContainer>
				<div style={{ gridColumn: "span 2" }}>
					<SendButton
						type="button"
						id="compileAndMatch"
						onClick={handleCompileRequest}
					>
						Compile {"→"}
					</SendButton>
				</div>
				<EditorContainer>
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
				</EditorContainer>
			</Editors>
			<ResultContainer>{children}</ResultContainer>
		</RootContainer>
	);
}

export default EditorView;
