import type { Value } from "@langlang/react";
import TreeNode from "./TreeNode";

// import { v4 as randomUuid } from "uuid";

interface TreeViewProps {
	tree: Value;
	renderLeafOnly?: boolean;
	parent?: string;
}

function TreeView({
	tree,
	renderLeafOnly = false,
	parent = "ROOT",
}: TreeViewProps) {
	switch (tree.type) {
		case "node":
			return (
				<TreeNode node={tree} renderLeafOnly={renderLeafOnly} parent={parent}>
					<TreeView
						tree={tree.expr}
						renderLeafOnly={renderLeafOnly}
						parent={`${parent}-${tree.name}`}
					/>
				</TreeNode>
			);
		case "sequence": {
			return (
				<TreeNode node={tree} renderLeafOnly={renderLeafOnly} parent={parent}>
					{tree.items.map((item, index) => (
						<TreeView
							tree={item}
							// biome-ignore lint/suspicious/noArrayIndexKey: idc!
							key={index}
							renderLeafOnly={renderLeafOnly}
							parent={`${parent}-${index}`}
						/>
					))}
				</TreeNode>
			);
		}

		case "error":
			return <TreeNode node={tree} renderLeafOnly={renderLeafOnly} parent={parent} />;
		default:
			return <TreeNode node={tree} renderLeafOnly={renderLeafOnly} parent={parent} />;
	}
}

export default TreeView;
