import TreeView from "./TreeView";
import type { LangLangValue } from "@langlang/react";
import { createContext, useRef, useState } from "react";
import { SourceLine, TreeViewContainer } from "./TreeExplorer.styles";

interface TreeExplorerProps {
	tree: LangLangValue;
}

interface TreeUiContextType {
	highlight: string | null;
	setHighlight: (highlight: string | null) => void;
	leafNodeSizeMap: Map<string, DOMRect>;
}

export const TreeUiContext = createContext<TreeUiContextType>({
	highlight: null,
	setHighlight: () => {},
	leafNodeSizeMap: new Map(),
});

function TreeExplorer({ tree }: TreeExplorerProps) {
	const [highlight, setHighlight] = useState<string | null>(null);
	const leafNodeSizeMapRef = useRef<Map<string, DOMRect>>(new Map());

	return (
		<TreeUiContext.Provider
			value={{
				highlight,
				setHighlight,
				leafNodeSizeMap: leafNodeSizeMapRef.current,
			}}
		>
			<TreeViewContainer onMouseLeave={() => setHighlight(null)}>
				<SourceLine>
					<TreeView tree={tree} renderLeafOnly />
				</SourceLine>
				<div>
					<TreeView tree={tree} />
				</div>
			</TreeViewContainer>
		</TreeUiContext.Provider>
	);
}

export default TreeExplorer;
