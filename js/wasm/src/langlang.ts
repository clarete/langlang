import type { Value } from "./types";

export type Config = Record<string, string | number | boolean>;

export type Range = { start: number; end: number };

export type NodeTypeTable = {
    String: number;
    Sequence: number;
    Node: number;
    Error: number;
};

export type RawResult<T> = { ok: true; value: T }
                         | { ok: false; error: string };



/**
 * createLanglang wraps the low-level Go/WASM bindings exposed at globalThis.langlang
 * into a more idiomatic OO API (Matcher/Tree/Node) with explicit dispose().
 */
export function createLanglang(raw: Raw = (globalThis as any).langlang): langlang {
    if (!raw) {
        throw new Error("langlang WASM bindings not found. Did you initialize the WASM module?");
    }
    return new langlang(raw);
}

export default class langlang {
    constructor(private raw: Raw) {}

    get NodeType(): NodeTypeTable {
        return this.raw.NodeType;
    }

    matcherFromString(grammar: string, cfg?: Config): Matcher {
        const mid = unwrap<{ id: number }>(this.raw.matcherFromString(String(grammar), cfg ?? undefined)).id;
        return new Matcher(this.raw, mid);
    }

    compileJson(grammar: string, input: string, cfg?: Config): Value {
        const m = this.matcherFromString(grammar, cfg);
        try {
            return m.match(input).value;
        } finally {
            try { m.dispose(); } catch (_) {}
        }
    }
}

interface Raw {
    NodeType: NodeTypeTable;
    matcherFromString(grammar: string, cfg?: Config): RawResult<{ id: number }>;
    freeMatcher(id: number): RawResult<void>;
    match(id: number, input: string): RawResult<{ consumed: number; value: Value }>;
}

const unwrap = <T>(res: any): T => {
    if (!res || res.ok !== true) {
        throw new Error(res?.error ?? "unknown langlang wasm error");
    }
    return res.value as T;
};

const canFinalize = typeof FinalizationRegistry === "function";

const matcherFinalizer: FinalizationRegistry<{ raw: Raw; id: number }> | null = canFinalize
    ? new FinalizationRegistry(({ raw, id }) => {
          try {
              unwrap<void>(raw.freeMatcher(id));
          } catch (_) {}
      })
    : null;

export class Matcher {
    private _disposed = false;

    constructor(
        private readonly raw: Raw,
        public readonly id: number,
    ) {
        matcherFinalizer?.register(this, { raw, id }, this);
    }

    match(input: string): { consumed: number; value: Value } {
        if (this._disposed) throw new Error("Matcher is disposed");
        return unwrap<{ consumed: number; value: Value }>(this.raw.match(this.id, String(input)));
    }

    dispose() {
        if (this._disposed) return;
        this._disposed = true;
        matcherFinalizer?.unregister(this);
        unwrap<void>(this.raw.freeMatcher(this.id));
    }
}
