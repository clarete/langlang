"use client";

import { useState } from "react";
import "./App.css";

// @ts-expect-error - wasmUrl is a file URL
import wasmUrl from "../langlang.wasm?url" with { type: "file" };
import { useWasmTest, type LangLangValue } from "@langlang/react";
import TreeView from "./TreeView";
import fixtures from "./fixtures";

import EditorView from "./EditorView";

function App() {
	const [result, setResult] = useState<LangLangValue | null>(null);
	const [format, setFormat] = useState<keyof typeof fixtures>("json");

	const [highlight, setHighlight] = useState<string | null>(null);

	if (!wasmUrl) {
		throw new Error("WASM URL is not set");
	}

	const { status, data, error } = useWasmTest(wasmUrl);

	const handleCompileJson = (__: string, input: string) => {
		if (!data) {
			return;
		}

		try {
			const result = data.compileJson(fixtures[format].grammar, input);
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
				<div style={{ marginBottom: "1rem" }}>
					<select
						defaultValue="json"
						onChange={(e) => setFormat(e.target.value as keyof typeof fixtures)}
					>
						<option value="demo">Demo</option>
						<option value="json">JSON</option>
						<option value="jsonStripped">JSON Stripped</option>
						<option value="csv">CSV</option>
						<option value="langlang">LangLang</option>
						<option value="xmlUnstable">XML Unstable</option>
					</select>
				</div>
				<EditorView
					grammar={fixtures[format].grammar}
					input={fixtures[format].input}
					onCompileRequest={handleCompileJson}
				>
					{result && (
						<div className="editors">
							<div className="tree-view-container-wrapper">
								<div className="tree-view-container">
									<div
										style={{
											display: "grid",
											gridTemplateColumns: "repeat(auto-fit, minmax(0, 1fr))",
											gap: "0.5rem",
										}}
										onMouseLeave={() => setHighlight(null)}
									>
										<TreeView
											tree={result}
											hideMeta
											highlight={highlight}
											setHighlight={setHighlight}
										/>
									</div>
									<div onMouseLeave={() => setHighlight(null)}>
										<TreeView
											tree={result}
											highlight={highlight}
											setHighlight={setHighlight}
										/>
									</div>
								</div>
							</div>
							<div className="response-container">
								<textarea
									className="response-area"
									value={JSON.stringify(result, null, 2)}
									rows={30}
								/>
							</div>
						</div>
					)}
				</EditorView>
			</>
		);
	}
}

export default App;
