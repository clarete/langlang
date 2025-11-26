declare class Go {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
}

declare class langlangWasm {
    compileJson(grammar: string, input: string): string;
    compileAndMatch(grammar: string, input: string): string;
}

