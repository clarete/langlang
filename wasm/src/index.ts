import LangLang from "./LangLang";

export class GoLangError<T = Error> extends Error {
    constructor(message: string, cause?: T) {
        super(message);
        this.name = "GoLangError";
        this.cause = cause as Error;
    }
}

export function initializeGoLang() {
    return new Promise<Go>((resolve, reject) => {

        try {
            const go = new Go();

            resolve(go);

        } catch (error) {
            reject(new GoLangError("Failed to initialize Go Lang", error));
        }

    });
}


export async function initializeLangLangWasm(langlangBinUrl: string) {
    const go = await initializeGoLang();
    const wasmRsp = await fetch(langlangBinUrl);

    if (!wasmRsp.ok) {
        throw new Error('Failed to fetch wasm file');
    }

    const webAssembly = await WebAssembly.instantiateStreaming(wasmRsp, go.importObject);
    go.run(webAssembly.instance);


    if (!('langlangWasm' in globalThis)) {
        throw new Error('langlangWasm not initialized');
    }

    //@ts-expect-error - langlangWasm is not typed
    return new LangLang(langlangWasm);
}

export * from "./Types";