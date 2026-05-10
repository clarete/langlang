import type { Value } from "@langlang/wasm";

export function makeRenderer(lang: string) {
    function render(value: Value, key: string | number): React.ReactNode {
        switch (value.type) {
            case "string":
                return value.value;
            case "sequence":
                return value.items.map((item, i) => render(item, i));
            case "node":
                return (
                    <span key={key} className={`${lang}-${value.name}`}>
                        {render(value.expr, 0)}
                    </span>
                );
            case "error":
                return value.expr ? (
                    <span key={key} className={`${lang}-error`}>
                        {render(value.expr, 0)}
                    </span>
                ) : null;
        }
    }
    return render;
}
