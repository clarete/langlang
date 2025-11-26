import type { LangLangValue } from "./Types";

class LangLang {
    constructor(private wasm: langlangWasm) {
    }

    compileAndMatch(grammar: string, input: string) {
        return this.wasm.compileAndMatch(grammar, input);
    }

    compileJson(grammar: string, input: string) {
        const astJsonString = this.wasm.compileJson(grammar, input);
        try {
            const astJson = JSON.parse(astJsonString);
            return astJson as LangLangValue;
        } catch (error) {

            return {
                type: "error",
                message: "Failed to parse JSON",
            } as LangLangValue;
        }
    }
}

export default LangLang;