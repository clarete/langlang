"use client";

import { initializeLangLangWasmFromAssets } from "@langlang/wasm/browser";
import { type AsyncState, useAsync } from "./utils/useAsync";
import type langlang from "../../wasm/dist/langlang";

export function useWasmTest(): AsyncState<langlang> {
  return useAsync(() => initializeLangLangWasmFromAssets());
}

export * from "@langlang/wasm";
