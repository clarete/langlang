import type { Value } from "@langlang/react";
import TraceNode from "./TraceNode";

// import { v4 as randomUuid } from "uuid";

interface TraceViewProps {
    tree: Value;
    renderLeafOnly?: boolean;
    parent?: string;
}

function TraceView({
    tree,
    renderLeafOnly = false,
    parent = "ROOT",
}: TraceViewProps) {
    switch (tree.type) {
        case "node":
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                >
                    <TraceView
                        tree={tree.expr}
                        renderLeafOnly={renderLeafOnly}
                        parent={`${parent}-${tree.name}`}
                    />
                </TraceNode>
            );
        case "sequence": {
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                >
                    {tree.items.map((item, index) => (
                        <TraceView
                            tree={item}
                            // biome-ignore lint/suspicious/noArrayIndexKey: idc!
                            key={index}
                            renderLeafOnly={renderLeafOnly}
                            parent={`${parent}-${index}`}
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
                />
            );
        default:
            return (
                <TraceNode
                    node={tree}
                    renderLeafOnly={renderLeafOnly}
                    parent={parent}
                />
            );
    }
}

export default TraceView;
