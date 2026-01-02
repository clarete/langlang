import type { Span, Value } from "@langlang/react";
import { useEffect, useMemo, useState } from "react";
import {
    CaretButton,
    Container,
    Label,
    Meta,
    PlaceholderCaret,
    Row,
} from "./TreeView.styles";

type Child = { key: string; value: Value; label?: string };

function getChildren(value: Value): Child[] {
    switch (value.type) {
        case "node":
            return [{ key: value.name, value: value.expr }];
        case "sequence":
            return value.items.map((item, i) => ({
                key: String(i),
                value: item,
            }));
        default:
            return [];
    }
}

function getLabel(value: Value): { label: string; meta?: string } {
    switch (value.type) {
        case "node":
            return { label: value.name, meta: "node" };
        case "sequence":
            return { label: "sequence", meta: `count=${value.count}` };
        case "string":
            return { label: `"${escapeString(value.value)}"`, meta: "string" };
        case "error":
            return {
                label: value.message ?? value.label ?? "error",
                meta: "error",
            };
        default: {
            // If this ever starts failing, the upstream `Value` union likely changed.
            const _exhaustive: never = value;
            return { label: String(_exhaustive), meta: "value" };
        }
    }
}

function escapeString(s: string): string {
    return s
        .replace(/"/g, '\\"')
        .replace(/\n/g, "\\n")
        .replace(/\r/g, "\\r")
        .replace(/\t/g, "\\t")
        .replace(/\\/g, "\\\\");
}

function computeDefaultExpandedPaths(root: Value, maxDepth: number): string[] {
    const expanded: string[] = [];
    const stack: Array<{ value: Value; path: string; depth: number }> = [
        { value: root, path: "ROOT", depth: 0 },
    ];
    while (stack.length) {
        const { value, path, depth } = stack.pop()!;
        if (depth >= maxDepth) {
            continue;
        }
        const children = getChildren(value);
        if (children.length === 0) {
            continue;
        }
        expanded.push(path);
        for (let i = children.length - 1; i >= 0; i--) {
            const child = children[i];
            const childPath = `${path}-${child.key}`;
            stack.push({
                value: child.value,
                path: childPath,
                depth: depth + 1,
            });
        }
    }
    return expanded;
}

interface TreeViewProps {
    tree: Value;
    defaultExpandDepth?: number;
    onHoverRange?: (span: Span | null) => void;
}

export default function TreeView({
    tree,
    defaultExpandDepth = 8,
    onHoverRange,
}: TreeViewProps) {
    const defaultExpandedPaths = useMemo(
        () => computeDefaultExpandedPaths(tree, defaultExpandDepth),
        [tree, defaultExpandDepth],
    );
    const [expanded, setExpanded] = useState<Set<string>>(
        () => new Set(defaultExpandedPaths),
    );
    useEffect(() => {
        setExpanded(new Set(defaultExpandedPaths));
    }, [defaultExpandedPaths]);

    const toggle = (path: string) => {
        setExpanded((prev) => {
            const next = new Set(prev);
            if (next.has(path)) next.delete(path);
            else next.add(path);
            return next;
        });
    };

    const renderNode = (value: Value, path: string, depth: number) => {
        const children = getChildren(value);
        const isExpandable = children.length > 0;
        const isExpanded = expanded.has(path);
        const { label, meta } = getLabel(value);

        return (
            <div key={path}>
                <Row
                    style={{ paddingLeft: `${depth * 14}px` }}
                    onMouseEnter={() => onHoverRange?.(value.span)}
                >
                    {isExpandable ? (
                        <CaretButton
                            type="button"
                            aria-label={isExpanded ? "Collapse" : "Expand"}
                            onClick={() => toggle(path)}
                        >
                            {isExpanded ? "▾" : "▸"}
                        </CaretButton>
                    ) : (
                        <PlaceholderCaret />
                    )}
                    <Label title={label}>{label}</Label>
                    {meta ? <Meta>{meta}</Meta> : null}
                </Row>
                {isExpandable && isExpanded
                    ? children.map((child) =>
                          renderNode(
                              child.value,
                              `${path}-${child.key}`,
                              depth + 1,
                          ),
                      )
                    : null}
            </div>
        );
    };

    return (
        <Container onMouseLeave={() => onHoverRange?.(null)}>
            {renderNode(tree, "ROOT", 0)}
        </Container>
    );
}
