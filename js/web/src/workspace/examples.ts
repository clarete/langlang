export interface PlaygroundFile {
    path: string;
    url: string;
}

export interface PlaygroundProject {
    label: string;
    grammars: PlaygroundFile[];
    inputs: PlaygroundFile[];
}

function loadUrl(path: string): string {
    return new URL(`../examples/${path}`, import.meta.url).href;
}

function declareProject(
    label: string,
    grammars: string[],
    inputs: string[],
): PlaygroundProject {
    return {
        label,
        grammars: grammars.map((p) => ({ path: p, url: loadUrl(p) })),
        inputs: inputs.map((p) => ({ path: p, url: loadUrl(p) })),
    };
}

export const defaultProjects = [
    declareProject(
        "JSON Stripped",
        ["json/json.stripped.peg"],
        ["json/small.json"],
    ),
    declareProject("JSON", ["json/json.peg"], ["json/small.json"]),
    declareProject(
        "CSV",
        ["csv/csv.peg"],
        ["csv/missions.csv", "csv/cities.csv"],
    ),
    declareProject("XML", ["xml/xml.peg"], ["xml/small.xml"]),
    declareProject(
        "Arithmetic Expressions",
        [
            "import/expr.peg",
            "import/value.peg",
            "import/number.peg",
            "import/string.peg",
        ],
        [
            "import/ok.txt",
            "import/string.txt",
            "import/override.txt",
            "import/bad.txt",
        ],
    ),
    declareProject(
        "PEG",
        ["peg/peg.peg"],
        ["json/json.stripped.peg", "peg/peg.peg"],
    ),
    declareProject(
        "langlang",
        ["langlang/langlang.peg"],
        [
            "csv/csv.peg",
            "xml/xml.peg",
            "json/json.peg",
            "json/json.stripped.peg",
            "peg/peg.peg",
            "langlang/langlang.peg",
        ],
    ),
];
