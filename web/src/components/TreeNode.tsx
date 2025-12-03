import type { LangLangValue } from "@langlang/react";
import type React from "react";
import { NodeContainer, NodeName, SequenceContainer } from "./TreeNode.styles";
import { useContext, useEffect, useRef } from "react";
import { TreeUiContext } from "./TreeExplorer";

interface TreeNodeProps {
	node: LangLangValue;
	children?: React.ReactNode;
	renderLeafOnly?: boolean;
	parent: string;
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

function TreeNode({
	node,
	children,
	renderLeafOnly = false,
	parent,
}: TreeNodeProps) {
	const { highlight, setHighlight, leafNodeSizeMap } =
		useContext(TreeUiContext);

	const nodeNameRef = useRef<HTMLDivElement>(null);

	// observe NodeName element size resize events
	useEffect(() => {
		const nodeNameElement = nodeNameRef.current;
		if (!renderLeafOnly && node.type === "string" && nodeNameElement) {
			leafNodeSizeMap.set(parent, nodeNameElement.getBoundingClientRect());

			// const observer = new ResizeObserver(() => {

			// });
			// observer.observe(nodeNameElement);
			// return () => observer.disconnect();
		}
	}, [renderLeafOnly, node, parent, leafNodeSizeMap]);

	if (renderLeafOnly && node.type !== "string") {
		return children;
	}

	const highlightProps = getHighlightProps(parent, highlight);

	switch (node.type) {
		case "string": {
			const sizeData = renderLeafOnly ? leafNodeSizeMap.get(parent) : undefined;

			const style = sizeData
				? {
						width: sizeData.width,
						paddingLeft: 0,
						paddingRight: 0,
					}
				: {};

			return (
				<NodeContainer className="string" data-parent={parent} leaf>
					<NodeName
						ref={nodeNameRef}
						style={style}
						{...highlightProps}
						onMouseEnter={() => setHighlight?.(parent)}
					>
						{node.value}
					</NodeName>
				</NodeContainer>
			);
		}
		case "sequence":
			return (
				<SequenceContainer
					data-parent={parent}
					style={{ "--count": node.count } as React.CSSProperties}
				>
					{children}
				</SequenceContainer>
			);
		case "node": {
			const sizeData = renderLeafOnly ? leafNodeSizeMap.get(parent) : undefined;
			const style = sizeData
				? {
						width: sizeData.width,
						paddingLeft: 0,
						paddingRight: 0,
					}
				: {};
			return (
				<NodeContainer data-parent={parent} style={style}>
					{!renderLeafOnly && (
						<NodeName
							ref={nodeNameRef}
							{...highlightProps}
							onMouseEnter={() => setHighlight?.(parent)}
						>
							{node.name}
						</NodeName>
					)}
					<div>{children}</div>
				</NodeContainer>
			);
		}

		default:
			return (
				<NodeContainer className="unknown" data-parent={parent}>
					{children}
				</NodeContainer>
			);
	}
}

export default TreeNode;
