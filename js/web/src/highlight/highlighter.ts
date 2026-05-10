import type { Value } from "@langlang/wasm";
import { getSharedWorkerClient, startSharedWorker } from "../worker/useLspWorker";
import { grammars } from "./grammars";

const matcherCache = new Map<string, Promise<number>>();

function getMatcherId(lang: string): Promise<number> {
    if (!matcherCache.has(lang)) {
        const entry = grammars[lang];
        matcherCache.set(
            lang,
            entry
                .load()
                .then((source) => startSharedWorker().then(() => source))
                .then((source) =>
                    getSharedWorkerClient()
                        .compile(source, entry.config)
                        .then(({ matcherId }) => matcherId),
                ),
        );
    }
    return matcherCache.get(lang)!;
}

export async function highlight(lang: string, code: string): Promise<Value> {
    const matcherId = await getMatcherId(lang);
    const { value } = await getSharedWorkerClient().match(matcherId, code);
    return value as Value;
}

export function isSupported(lang: string): boolean {
    return lang in grammars;
}
