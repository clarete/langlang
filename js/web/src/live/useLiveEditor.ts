import { useEffect, useRef, useState } from "react";
import { useLspWorker } from "../worker/useLspWorker";
import type { Value, Span } from "@langlang/wasm";
import { useDebouncedEffect } from "./useDebouncedEffect";

export interface LiveEditorSettings {
    captureSpaces?: boolean;
    handleSpaces?: boolean;
    enableInline?: boolean;
    showFails?: boolean;
}

export interface UseLiveEditorOptions {
    grammar: string;
    input: string;
    settings?: LiveEditorSettings;
}

export interface UseLiveEditorResult {
    result: Value | null;
    outputError: string | null;
    hoverRange: Span | null;
    setHoverRange: (span: Span | null) => void;
    workerStatus: "pending" | "ready" | "error";
}

const COMPILATION_DEBOUNCE_MS = 200;
const MATCHING_DEBOUNCE_MS = 50;

function formatError(error: unknown): string {
    if (error instanceof Error) return error.message;
    return String(error);
}

export function useLiveEditor(opts: UseLiveEditorOptions): UseLiveEditorResult {
    const { grammar, input, settings } = opts;
    const { status, client } = useLspWorker();

    const [result, setResult] = useState<Value | null>(null);
    const [outputError, setOutputError] = useState<string | null>(null);
    const [hoverRange, setHoverRange] = useState<Span | null>(null);
    const [matcherVersion, setMatcherVersion] = useState(0);
    const [isCompiling, setIsCompiling] = useState(false);

    const matcherIdRef = useRef<number | null>(null);
    const compileSeqRef = useRef(0);

    const matcherCfg: Record<string, unknown> = {
        "grammar.capture_spaces": settings?.captureSpaces ?? false,
        "grammar.handle_spaces": settings?.handleSpaces ?? true,
        "compiler.inline.enabled": settings?.enableInline ?? true,
        "vm.show_fails": settings?.showFails ?? true,
    };

    useDebouncedEffect(
        () => {
            if (!client) return;
            compileSeqRef.current += 1;
            const seq = compileSeqRef.current;
            setIsCompiling(true);
            setOutputError(null);

            if (matcherIdRef.current !== null) {
                client.freeMatcher(matcherIdRef.current);
                matcherIdRef.current = null;
            }

            client
                .compile(grammar, matcherCfg)
                .then(({ matcherId }) => {
                    if (compileSeqRef.current !== seq) {
                        client.freeMatcher(matcherId);
                        return;
                    }
                    matcherIdRef.current = matcherId;
                    setOutputError(null);
                    setMatcherVersion((v) => v + 1);
                })
                .catch((e) => {
                    if (compileSeqRef.current !== seq) return;
                    setResult(null);
                    setOutputError(formatError(e));
                })
                .finally(() => {
                    if (compileSeqRef.current === seq) setIsCompiling(false);
                });
        },
        [client, grammar, settings?.captureSpaces, settings?.handleSpaces, settings?.enableInline, settings?.showFails],
        COMPILATION_DEBOUNCE_MS,
    );

    useDebouncedEffect(
        () => {
            if (isCompiling) { setOutputError(null); return; }
            if (!client) return;
            const matcherId = matcherIdRef.current;
            if (matcherId === null) return;

            client
                .match(matcherId, input)
                .then(({ value }) => {
                    setResult(value as Value);
                    setOutputError(null);
                })
                .catch((e) => {
                    setResult(null);
                    setOutputError(formatError(e));
                });
        },
        [client, input, matcherVersion, isCompiling],
        MATCHING_DEBOUNCE_MS,
    );

    useEffect(() => {
        return () => {
            if (matcherIdRef.current !== null && client) {
                client.freeMatcher(matcherIdRef.current);
                matcherIdRef.current = null;
            }
        };
    }, [client]);

    return { result, outputError, hoverRange, setHoverRange, workerStatus: status };
}
