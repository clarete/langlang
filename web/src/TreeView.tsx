import type { LangLangValue } from "@langlang/react";
import React from "react";
import { NodeContainer, NodeName, SequenceContainer } from "./TreeView.styles";

// import { v4 as randomUuid } from "uuid";

interface PreviewNodeProps {
	node: LangLangValue;
	children?: React.ReactNode;
	hideMeta?: boolean;
	parent: string;
	highlight?: string | null;
	setHighlight?: (highlight: string | null) => void;
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

function PreviewNode({
	node,
	children,
	hideMeta = false,
	parent,
	highlight,
	setHighlight,
}: PreviewNodeProps) {
	if (hideMeta && node.type !== "string") {
		return children;
	}

	const highlightProps = getHighlightProps(parent, highlight);

	switch (node.type) {
		case "string":
			return (
				<NodeContainer className="string" data-parent={parent}>
					<NodeName
						{...highlightProps}
						onMouseEnter={() => setHighlight?.(parent)}
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
					{!hideMeta && (
						<NodeName
							{...highlightProps}
							onMouseEnter={() => setHighlight?.(parent)}
						>
							{node.name}
						</NodeName>
					)}
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

interface TreeViewProps {
	tree: LangLangValue;
	hideMeta?: boolean;
	parent?: string;
	highlight?: string | null;
	setHighlight?: (highlight: string | null) => void;
}

function TreeView({
	tree,
	hideMeta = false,
	parent = "ROOT",
	highlight,
	setHighlight,
}: TreeViewProps) {
	switch (tree.type) {
		case "node":
			return (
				<PreviewNode
					node={tree}
					hideMeta={hideMeta}
					parent={parent}
					highlight={highlight}
					setHighlight={setHighlight}
				>
					<TreeView
						tree={tree.expr}
						hideMeta={hideMeta}
						parent={`${parent}-${tree.name}`}
						highlight={highlight}
						setHighlight={setHighlight}
					/>
				</PreviewNode>
			);
		case "sequence": {
			return (
				<PreviewNode
					node={tree}
					hideMeta={hideMeta}
					parent={parent}
					highlight={highlight}
					setHighlight={setHighlight}
				>
					{tree.items.map((item, index) => (
						<TreeView
							tree={item}
							// biome-ignore lint/suspicious/noArrayIndexKey: idc!
							key={index}
							hideMeta={hideMeta}
							parent={`${parent}-${index}`}
							highlight={highlight}
							setHighlight={setHighlight}
						/>
					))}
				</PreviewNode>
			);
		}

		case "error":
			return (
				<PreviewNode
					node={tree}
					hideMeta={hideMeta}
					parent={parent}
					highlight={highlight}
					setHighlight={setHighlight}
				/>
			);
		default:
			return (
				<PreviewNode
					node={tree}
					hideMeta={hideMeta}
					parent={parent}
					highlight={highlight}
					setHighlight={setHighlight}
				/>
			);
	}
}

export default TreeView;
