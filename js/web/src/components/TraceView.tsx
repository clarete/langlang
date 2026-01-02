import type { Span, Value } from "@langlang/react";
import TraceNode from "./TraceNode";

// import { v4 as randomUuid } from "uuid";

interface TraceViewProps {
    tree: Value;
    renderLeafOnly?: boolean;
    parent?: string;
    onHoverRange?: (span: Span | null) => void;
}

function TraceView({
    tree,
    renderLeafOnly = false,
    parent = "ROOT",
    onHoverRange,
}: TraceViewProps) {
    switch (tree.type) {
        case "node":
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                    onHoverRange={onHoverRange}
                >
                    <TraceView
                        tree={tree.expr}
                        renderLeafOnly={renderLeafOnly}
                        parent={`${parent}-${tree.name}`}
                        onHoverRange={onHoverRange}
                    />
                </TraceNode>
            );
        case "sequence": {
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                    onHoverRange={onHoverRange}
                >
                    {tree.items.map((item, index) => (
                        <TraceView
                            tree={item}
                            // biome-ignore lint/suspicious/noArrayIndexKey: idc!
                            key={index}
                            renderLeafOnly={renderLeafOnly}
                            parent={`${parent}-${index}`}
                            onHoverRange={onHoverRange}
                        />
                    ))}
                </TraceNode>
            );
        }

        case "error":
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                    onHoverRange={onHoverRange}
                />
            );
        default:
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                    onHoverRange={onHoverRange}
                />
            );
    }
}

export default TraceView;
