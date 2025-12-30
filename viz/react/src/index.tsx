"use client";

import { initializeLangLangWasm } from "@langlang/wasm";
import { useState, useEffect, use } from "react";
import { type AsyncState, useAsync } from "./utils/useAsync";
import type LangLang from "../../wasm/dist/LangLang";

export function useWasmTest(langlangWasmUrl: string): AsyncState<LangLang> {
	const request = useAsync(() => initializeLangLangWasm(langlangWasmUrl));

	return request;
}

export * from "@langlang/wasm";
