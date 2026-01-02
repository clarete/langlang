export type LanguageKey =
    | "json"
    | "jsonStripped"
    | "csv"
    | "xmlUnstable"
    | "langlang"
    | "protoCirc"
    | "tiny";

export interface PlaygroundPair {
    id: LanguageKey;
    label: string;
    grammar: string;
    input: string;
}

function declareLanguage(label: string, key: LanguageKey): PlaygroundPair {
    const grammar = new URL(`${key}.peg`, import.meta.url).href;
    const input = new URL(`${key}.${key}`, import.meta.url).href;

    return {
        id: key,
        label,
        grammar,
        input,
    };
}

export const playgroundPairs = {
    tiny: declareLanguage("Demo (Tiny)", "tiny"),
    json: declareLanguage("JSON", "json"),
    jsonStripped: declareLanguage("JSON Stripped", "jsonStripped"),
    csv: declareLanguage("CSV", "csv"),
    xmlUnstable: declareLanguage("XML Unstable", "xmlUnstable"),
    langlang: declareLanguage("LangLang", "langlang"),
    protoCirc: declareLanguage("Proto Circ", "protoCirc"),
} as const;

export const playgroundPairsKeys = Object.keys(
    playgroundPairs,
) as LanguageKey[];
export const playgroundPairsList = Object.values(playgroundPairs);
