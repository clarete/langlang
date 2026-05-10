import TraceView from "./TraceView";
import type { Span, Value } from "@langlang/wasm";
import { createContext, useCallback, useRef, useState } from "react";
import { SourceLine, TraceViewContainer } from "./TraceExplorer.styles";

interface TraceExplorerProps {
    tree: Value;
    onHoverRange?: (span: Span | null) => void;
}

interface TraceUiContextType {
    highlight: string | null;
    setHighlight: (highlight: string | null) => void;
    leafNodeSizeMap: Map<string, DOMRect>;
    notifyMeasured: () => void;
}

export const TraceUiContext = createContext<TraceUiContextType>({
    highlight: null,
    setHighlight: () => {},
    leafNodeSizeMap: new Map(),
    notifyMeasured: () => {},
});

function TraceExplorer({ tree, onHoverRange }: TraceExplorerProps) {
    const [highlight, setHighlight] = useState<string | null>(null);
    const leafNodeSizeMapRef = useRef<Map<string, DOMRect>>(new Map());
    const [, setSizeVersion] = useState(0);
    const measurePending = useRef(false);

    const notifyMeasured = useCallback(() => {
        if (!measurePending.current) {
            measurePending.current = true;
            requestAnimationFrame(() => {
                setSizeVersion((v) => v + 1);
                measurePending.current = false;
            });
        }
    }, []);

    return (
        <TraceUiContext.Provider
            value={{
                highlight,
                setHighlight,
                leafNodeSizeMap: leafNodeSizeMapRef.current,
                notifyMeasured,
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
