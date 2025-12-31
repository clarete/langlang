import type { Monaco } from "@monaco-editor/react";

const PEG_LANGUAGE_ID = "peg";

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

                // char class
                [/\[/, { token: "delimiter.square", bracket: "@open", next: "@charclass" }],

                // strings
                [/"/, { token: "string.quote", bracket: "@open", next: "@dstring" }],
                [/'/, { token: "string.quote", bracket: "@open", next: "@sstring" }],

                // operators / delimiters
                // NOTE: Monaco's default `vs-dark` theme renders plain `operator`
                // close to normal foreground, so we use `keyword.operator` for
                // better contrast.
                [/<-/, "keyword.operator"],
                [/\/(?!\/)/, "keyword.operator"], // prevent collision with `//` comments
                [/[?*+]/, "keyword.operator"],
                [/([&!#]|[.])/,
                    {
                        cases: {
                            "@default": "keyword.operator",
                        },
                    },
                ],
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
                // If anything else happens before `<-`, give up and return to normal lexing.
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
                [/"/, { token: "string.quote", bracket: "@close", next: "@pop" }],
            ],
            sstring: [
                [/[^\\']+/, "string"],
                [/\\./, "string.escape"],
                [/'/, { token: "string.quote", bracket: "@close", next: "@pop" }],
            ],
            charclass: [
                [/[^\\\]]+/, "string"],
                [/\\./, "string.escape"],
                [/\]/, { token: "delimiter.square", bracket: "@close", next: "@pop" }],
            ],
        },
    });
}
