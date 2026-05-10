type GrammarEntry = {
    load: () => Promise<string>;
    config: Record<string, unknown>;
};

const defaultConfig = {
    "grammar.capture_spaces": true,
    "grammar.handle_spaces": true,
    "compiler.inline.enabled": true,
    "vm.show_fails": false,
};

export const grammars: Record<string, GrammarEntry> = {
    peg: {
        load: () => import("../../../../grammars/langlang.peg?raw").then((m) => m.default),
        config: defaultConfig,
    },
    go: {
        load: () => import("../../../../grammars/go.peg?raw").then((m) => m.default),
        config: { ...defaultConfig, "grammar.handle_spaces": false },
    },
};
