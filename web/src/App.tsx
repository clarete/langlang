"use client";
// import { useWasmTest } from "@langlang/react";
import { useRef, useState } from "react";
import "./App.css";

// @ts-expect-error - wasmUrl is a file URL
import wasmUrl from "../langlang.wasm?url" with { type: "file" };
import { useWasmTest, type LangLangValue } from "@langlang/react";
import TreeView from "./TreeView";

const DEBUG_GRAMMAR = `
Expr     <- Multi (( '+' / '-' ) Multi)*
Multi    <- Primary (( '*' / '/' ) Primary)*
Primary  <- Call / Id / Num / '(' Expr ')'

Call     <- Id '(' Params ')'
Params   <- (Expr (',' Expr)*)?

Num      <- [1-9][0-9]* / '0'
Id       <- [a-zA-Z_][a-zA-Z0-9_]*
`;

const DEBUG_INPUT = `
he+1
`;

function App() {
	const [count, setCount] = useState(0);
	const [result, setResult] = useState<LangLangValue | null>(null);
	const grammarRef = useRef<HTMLTextAreaElement>(null);
	const inputRef = useRef<HTMLTextAreaElement>(null);
	const [highlight, setHighlight] = useState<string | null>(null);

	if (!wasmUrl) {
		throw new Error("WASM URL is not set");
	}

	const { status, data, error } = useWasmTest(wasmUrl);

	const handleCompileJson = () => {
		const grammar = grammarRef.current?.value;
		const input = inputRef.current?.value;
		if (!grammar || !input || !data) {
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

	if (status === "pending") {
		return <div>Loading...</div>;
	}

	if (status === "error") {
		return <div>Error: {error?.message}</div>;
	}

	if (status === "success") {
		return (
			<>
				<div className="playground-container">
					<label htmlFor="grammar">Grammar</label>
					<textarea
						defaultValue={DEBUG_GRAMMAR}
						id="grammar"
						className="response-area"
						placeholder="Grammar"
						rows={10}
						ref={grammarRef}
					/>
					<label htmlFor="input">Input</label>
					<textarea
						defaultValue={DEBUG_INPUT}
						className="response-area"
						id="input"
						placeholder="Input"
						rows={10}
						ref={inputRef}
					/>
					<div style={{ gridColumn: "span 2" }}>
						<button
							type="button"
							id="compileAndMatch"
							onClick={handleCompileJson}
							className="send-button"
						>
							Compile JSON
						</button>
					</div>
				</div>
				{result && (
					<>
						<div
							style={{
								display: "grid",
								gridTemplateColumns: "repeat(auto-fit, minmax(0, 1fr))",
								gap: "0.5rem",
							}}
						>
							<TreeView
								tree={result}
								hideMeta
								highlight={highlight}
								setHighlight={setHighlight}
							/>
						</div>
						<TreeView
							tree={result}
							highlight={highlight}
							setHighlight={setHighlight}
						/>
						<textarea
							className="response-area"
							value={JSON.stringify(result, null, 2)}
							rows={30}
						/>
					</>
				)}
			</>
		);
	}

	return (
		<>
			<h1>@langlang/web-test</h1>
			<div className="card">
				<button onClick={() => setCount((count) => count + 1)} type="button">
					count is {count}
				</button>
				{/* <p></p> */}
			</div>
		</>
	);
}

export default App;
