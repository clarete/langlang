import TraceView from "./TraceView";
import type { Span, Value } from "@langlang/wasm";
import { createContext, useState } from "react";
import { TraceViewContainer } from "./TraceExplorer.styles";

interface TraceExplorerProps {
    tree: Value;
    onHoverRange?: (span: Span | null) => void;
}

interface TraceUiContextType {
    highlight: string | null;
    setHighlight: (highlight: string | null) => void;
}

export const TraceUiContext = createContext<TraceUiContextType>({
    highlight: null,
    setHighlight: () => {},
});

function TraceExplorer({ tree, onHoverRange }: TraceExplorerProps) {
    const [highlight, setHighlight] = useState<string | null>(null);

    return (
        <TraceUiContext.Provider value={{ highlight, setHighlight }}>
            <TraceViewContainer
                onMouseLeave={() => {
                    setHighlight(null);
                    onHoverRange?.(null);
                }}
            >
                <div>
                    <TraceView tree={tree} onHoverRange={onHoverRange} />
                </div>
            </TraceViewContainer>
        </TraceUiContext.Provider>
    );
}

export default TraceExplorer;
