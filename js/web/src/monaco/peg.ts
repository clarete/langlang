import type { Monaco } from "@monaco-editor/react";
import { pegColors } from "../highlight/pegColors";

const PEG_LANGUAGE_ID = "peg";
export const PEG_THEME_ID = "langlang-dark";
export const PEG_LIGHT_THEME_ID = "langlang-light";

// Strip '#' for Monaco theme rules which expect bare hex strings.
function hex(color: string): string {
    return color.startsWith("#") ? color.slice(1) : color;
}

function isPegRegistered(monaco: Monaco): boolean {
    return monaco.languages
        .getLanguages()
        .some((lang: { id: string }) => lang.id === PEG_LANGUAGE_ID);
}

export function registerPegLanguage(monaco: Monaco) {
    if (isPegRegistered(monaco)) {
        return;
    }
    monaco.languages.register({
        id: PEG_LANGUAGE_ID,
        extensions: [".peg"],
        aliases: ["PEG", "peg"],
        mimetypes: ["text/x-peg"],
    });
    monaco.languages.setLanguageConfiguration(PEG_LANGUAGE_ID, {
        comments: {
            lineComment: "//",
        },
        brackets: [
            ["{", "}"],
            ["[", "]"],
            ["(", ")"],
        ],
        autoClosingPairs: [
            { open: "{", close: "}" },
            { open: "[", close: "]" },
            { open: "(", close: ")" },
            { open: '"', close: '"', notIn: ["string"] },
            { open: "'", close: "'", notIn: ["string"] },
        ],
        surroundingPairs: [
            { open: "{", close: "}" },
            { open: "[", close: "]" },
            { open: "(", close: ")" },
            { open: '"', close: '"' },
            { open: "'", close: "'" },
        ],
    });
    monaco.languages.setMonarchTokensProvider(PEG_LANGUAGE_ID, {
        defaultToken: "",
        tokenPostfix: ".peg",

        brackets: [
            { open: "{", close: "}", token: "delimiter.curly" },
            { open: "[", close: "]", token: "delimiter.square" },
            { open: "(", close: ")", token: "delimiter.parenthesis" },
        ],

        keywords: ["@import"],

        tokenizer: {
            root: [
                { include: "@whitespace" },

                [/[@]import\b/, "keyword"],

                // rule name at start of line: `Name <-`
                [/^\s*[a-zA-Z_][a-zA-Z0-9_]*(?=\s*<-\s*)/, "type.identifier"],

                // rule name on its own line (allow the `<-` to appear on a following line)
                [
                    /^\s*[a-zA-Z_][a-zA-Z0-9_]*\s*(?=$|\/\/|\/\*)/,
                    { token: "type.identifier", next: "@afterRuleName" },
                ],

                // identifiers elsewhere
                [/[a-zA-Z_][a-zA-Z0-9_]*/, "identifier"],

                // label/annotation marker: `^label`
                [/\^[a-zA-Z_][a-zA-Z0-9_]*/, "annotation"],

                // char class — use constant.character inside so it can be
                // colored differently from string literal content
                [
                    /\[/,
                    {
                        token: "delimiter.square",
                        bracket: "@open",
                        next: "@charclass",
                    },
                ],

                // strings
                [
                    /"/,
                    {
                        token: "string.quote",
                        bracket: "@open",
                        next: "@dstring",
                    },
                ],
                [
                    /'/,
                    {
                        token: "string.quote",
                        bracket: "@open",
                        next: "@sstring",
                    },
                ],

                // operators
                [/<-/, "keyword.operator"],
                [/\/(?!\/)/, "keyword.operator"],
                [/[?*+]/, "keyword.operator"],
                [/([&!#]|[.])/, "keyword.operator"],
                [/[{}()[\],:]/, "delimiter"],
            ],
            whitespace: [
                [/[ \t\r\n]+/, ""],
                [/\/\/.*$/, "comment"],
                [/\/\*/, "comment", "@comment"],
            ],
            afterRuleName: [
                { include: "@whitespace" },
                [/<-/, "keyword.operator", "@pop"],
                [/./, "", "@pop"],
            ],
            comment: [
                [/[^/*]+/, "comment"],
                [/\*\//, "comment", "@pop"],
                [/[/\*]/, "comment"],
            ],
            dstring: [
                [/[^\\"]+/, "string"],
                [/\\./, "string.escape"],
                [
                    /"/,
                    { token: "string.quote", bracket: "@close", next: "@pop" },
                ],
            ],
            sstring: [
                [/[^\\']+/, "string"],
                [/\\./, "string.escape"],
                [
                    /'/,
                    { token: "string.quote", bracket: "@close", next: "@pop" },
                ],
            ],
            // Use constant.character (not string) so charclass content gets
            // its own distinct color in the theme.
            charclass: [
                [/[^\\\]]+/, "constant.character"],
                [/\\./, "constant.character"],
                [
                    /\]/,
                    {
                        token: "delimiter.square",
                        bracket: "@close",
                        next: "@pop",
                    },
                ],
            ],
        },
    });

    const tokenRules = [
        { token: "type.identifier.peg",     foreground: hex(pegColors.ruleName) },
        { token: "identifier.peg",           foreground: hex(pegColors.ruleRef) },
        { token: "string.peg",               foreground: hex(pegColors.literal) },
        { token: "string.quote.peg",         foreground: hex(pegColors.literal) },
        { token: "string.escape.peg",        foreground: hex(pegColors.literal) },
        { token: "constant.character.peg",   foreground: hex(pegColors.charClass) },
        { token: "delimiter.square.peg",     foreground: hex(pegColors.charClass) },
        { token: "keyword.peg",              foreground: hex(pegColors.label) },
        { token: "annotation.peg",           foreground: hex(pegColors.label) },
        { token: "comment.peg",              foreground: hex(pegColors.comment) },
        { token: "keyword.operator.peg",     foreground: hex(pegColors.operator) },
    ];

    monaco.editor.defineTheme(PEG_THEME_ID, {
        base: "vs-dark",
        inherit: true,
        rules: tokenRules,
        colors: {},
    });

    monaco.editor.defineTheme(PEG_LIGHT_THEME_ID, {
        base: "vs",
        inherit: true,
        rules: tokenRules,
        colors: { "editor.background": "#f5f4f1" },
    });
}
