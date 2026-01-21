import TraceView from "./TraceView";
import type { Span, Value } from "@langlang/wasm";
import { createContext, useRef, useState } from "react";
import { SourceLine, TraceViewContainer } from "./TraceExplorer.styles";

interface TraceExplorerProps {
    tree: Value;
    onHoverRange?: (span: Span | null) => void;
}

interface TraceUiContextType {
    highlight: string | null;
    setHighlight: (highlight: string | null) => void;
    leafNodeSizeMap: Map<string, DOMRect>;
}

export const TraceUiContext = createContext<TraceUiContextType>({
    highlight: null,
    setHighlight: () => {},
    leafNodeSizeMap: new Map(),
});

function TraceExplorer({ tree, onHoverRange }: TraceExplorerProps) {
    const [highlight, setHighlight] = useState<string | null>(null);
    const leafNodeSizeMapRef = useRef<Map<string, DOMRect>>(new Map());

    return (
        <TraceUiContext.Provider
            value={{
                highlight,
                setHighlight,
                leafNodeSizeMap: leafNodeSizeMapRef.current,
            }}
        >
            <TraceViewContainer
                onMouseLeave={() => {
                    setHighlight(null);
                    onHoverRange?.(null);
                }}
            >
                <SourceLine>
                    <TraceView
                        tree={tree}
                        renderLeafOnly
                        onHoverRange={onHoverRange}
                    />
                </SourceLine>
                <div>
                    <TraceView tree={tree} onHoverRange={onHoverRange} />
                </div>
            </TraceViewContainer>
        </TraceUiContext.Provider>
    );
}

export default TraceExplorer;
