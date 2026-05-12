import type { Span, Value } from "@langlang/wasm";
import type React from "react";
import { NodeContainer, NodeName, SequenceContainer } from "./TraceNode.styles";
import { useContext } from "react";
import { TraceUiContext } from "./TraceExplorer";

interface TraceNodeProps {
    node: Value;
    children?: React.ReactNode;
    parent: string;
    onHoverRange?: (span: Span | null) => void;
}

const clamp = (value: number, min: number, max: number): number =>
    Math.max(min, Math.min(value, max));

function getHighlightProps(
    parent: string,
    highlight?: string | null,
): { highlighted: boolean; parentHighlighted: boolean; level: number } {
    if (!highlight)
        return { highlighted: false, parentHighlighted: false, level: 0 };

    const isHighlighted = highlight.startsWith(parent);
    const previousParent = parent.split("-").slice(0, -1).join("-");
    const isParentHighlighted = previousParent.startsWith(highlight);

    const parentParts = parent.split("-");
    const highlightParts = highlight.split("-");

    if (isParentHighlighted) {
        return {
            highlighted: false,
            parentHighlighted: true,
            level: clamp(parentParts.length - highlightParts.length, 0, 4),
        };
    }

    if (isHighlighted) {
        let level = 0;
        const total = highlightParts.length;
        for (let i = 0; i < highlightParts.length; i++) {
            if (parentParts[i] !== highlightParts[i]) {
                break;
            }
            level++;
        }
        return {
            highlighted: true,
            parentHighlighted: false,
            level: clamp(total - level, 0, 4),
        };
    }

    return { highlighted: false, parentHighlighted: false, level: 0 };
}

function TraceNode({ node, children, parent, onHoverRange }: TraceNodeProps) {
    const { highlight, setHighlight } = useContext(TraceUiContext);

    const highlightProps = getHighlightProps(parent, highlight);

    switch (node.type) {
        case "string":
            return (
                <NodeContainer className="string" data-parent={parent} leaf>
                    <NodeName
                        {...highlightProps}
                        onMouseEnter={() => {
                            setHighlight?.(parent);
                            onHoverRange?.(node.span);
                        }}
                    >
                        {node.value}
                    </NodeName>
                </NodeContainer>
            );
        case "sequence":
            return (
                <SequenceContainer
                    data-parent={parent}
                    style={{ "--count": node.count } as React.CSSProperties}
                >
                    {children}
                </SequenceContainer>
            );
        case "node":
            return (
                <NodeContainer data-parent={parent}>
                    <NodeName
                        {...highlightProps}
                        onMouseEnter={() => {
                            setHighlight?.(parent);
                            onHoverRange?.(node.span);
                        }}
                    >
                        {node.name}
                    </NodeName>
                    <div>{children}</div>
                </NodeContainer>
            );
        default:
            return (
                <NodeContainer className="unknown" data-parent={parent}>
                    {children}
                </NodeContainer>
            );
    }
}

export default TraceNode;
