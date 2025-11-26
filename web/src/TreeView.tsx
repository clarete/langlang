import type { LangLangValue } from "@langlang/react";
// import { v4 as randomUuid } from "uuid";

interface PreviewNodeProps {
	node: LangLangValue;
	children?: React.ReactNode;
	hideMeta?: boolean;
	parent: string;
	highlight?: string | null;
	setHighlight?: (highlight: string | null) => void;
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

	const isHighlighted = highlight?.startsWith(parent);

	switch (node.type) {
		case "string":
			return (
				<div className="node string" data-parent={parent}>
					<div
						className={`node-name ${isHighlighted ? "highlighted" : ""}`}
						onMouseEnter={() => setHighlight?.(parent)}
					>
						{node.value}
					</div>
				</div>
			);
		case "sequence":
			return (
				<div
					data-parent={parent}
					className="sequence"
					style={{ "--count": node.count } as React.CSSProperties}
				>
					{children}
				</div>
			);
		case "node":
			return (
				<div className="node" data-parent={parent}>
					{!hideMeta && (
						<div
							className={`node-name ${isHighlighted ? "highlighted" : ""}`}
							onMouseEnter={() => setHighlight?.(parent)}
						>
							{node.name}
						</div>
					)}
					<div>{children}</div>
				</div>
			);

		default:
			return (
				<div className="node unknown" data-parent={parent}>
					{children}
				</div>
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
