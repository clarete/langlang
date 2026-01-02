import { Simplify } from "type-fest";

export type LanguageKey =
    | "json"
    | "json.stripped"
    | "csv"
    | "xml"
    | "langlang"
    | "protoCirc"
    | "tiny";

export interface PlaygroundPair {
    id: LanguageKey;
    label: string;
    grammar: string;
    input: string;
}

function declareLanguage(
    label: string,
    key: LanguageKey,
    cwd = "grammars",
): PlaygroundPair {
    const grammar = new URL(`${cwd}/${key}.peg`, import.meta.url).href;
    const input = new URL(`${cwd}/${key}.${key}`, import.meta.url).href;

    return {
        id: key,
        label,
        grammar,
        input,
    };
}

export const playgroundPairs: Simplify<Record<LanguageKey, PlaygroundPair>> = {
    tiny: declareLanguage("Demo (Tiny)", "tiny"),
    json: declareLanguage("JSON", "json"),
    "json.stripped": declareLanguage("JSON Stripped", "json.stripped"),
    csv: declareLanguage("CSV", "csv"),
    xml: declareLanguage("XML Unstable", "xml", "grammars/unstable"),
    langlang: declareLanguage("LangLang", "langlang"),
    protoCirc: declareLanguage("Proto Circ", "protoCirc", "grammars/jeff"),
};

export const playgroundPairsKeys = Object.keys(
    playgroundPairs,
) as LanguageKey[];
export const playgroundPairsList = Object.values(playgroundPairs);
