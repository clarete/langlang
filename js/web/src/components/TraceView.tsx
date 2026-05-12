import type { Span, Value } from "@langlang/wasm";
import TraceNode from "./TraceNode";

interface TraceViewProps {
    tree: Value;
    parent?: string;
    onHoverRange?: (span: Span | null) => void;
}

function TraceView({
    tree,
    parent = "ROOT",
    onHoverRange,
}: TraceViewProps) {
    switch (tree.type) {
        case "node":
            return (
                <TraceNode
                    node={tree}
                    parent={parent}
                    onHoverRange={onHoverRange}
                >
                    <TraceView
                        tree={tree.expr}
                        parent={`${parent}-${tree.name}`}
                        onHoverRange={onHoverRange}
                    />
                </TraceNode>
            );
        case "sequence":
            return (
                <TraceNode
                    node={tree}
                    parent={parent}
                    onHoverRange={onHoverRange}
                >
                    {tree.items.map((item, index) => (
                        <TraceView
                            tree={item}
                            // biome-ignore lint/suspicious/noArrayIndexKey: idc!
                            key={index}
                            parent={`${parent}-${index}`}
                            onHoverRange={onHoverRange}
                        />
                    ))}
                </TraceNode>
            );
        default:
            return (
                <TraceNode
                    node={tree}
                    parent={parent}
                    onHoverRange={onHoverRange}
                />
            );
    }
}

export default TraceView;
