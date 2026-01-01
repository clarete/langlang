import TraceView from "./TraceView";
import type { Value } from "@langlang/react";
import { createContext, useRef, useState } from "react";
import { SourceLine, TraceViewContainer } from "./TraceExplorer.styles";

interface TraceExplorerProps {
    tree: Value;
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

function TraceExplorer({ tree }: TraceExplorerProps) {
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
            <TraceViewContainer onMouseLeave={() => setHighlight(null)}>
                <SourceLine>
                    <TraceView tree={tree} renderLeafOnly />
                </SourceLine>
                <div>
                    <TraceView tree={tree} />
                </div>
            </TraceViewContainer>
        </TraceUiContext.Provider>
    );
}

export default TraceExplorer;
