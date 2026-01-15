import type {
    WorkspaceDirNode,
    WorkspaceExampleNode,
    WorkspaceNode,
    WorkspaceNodeId,
    WorkspaceStateV1,
} from "./types";

const STORAGE_KEY = "langlang.workspace.v1";

export function createDefaultWorkspace(): WorkspaceStateV1 {
    const root: WorkspaceDirNode = {
        type: "dir",
        id: "root",
        name: "My Workspace",
        children: [
            {
                type: "grammar",
                id: "scratch",
                name: "scratch",
                project: {
                    entry: "grammar.peg",
                    files: [{ path: "grammar.peg", content: "" }],
                    activePath: "grammar.peg",
                },
                inputs: [{ path: "input.txt", content: "" }],
                activeInputPath: "input.txt",
            },
        ],
    };
    return {
        version: 1,
        root,
        lastSelection: { kind: "user", id: "scratch" },
    };
}

export function loadWorkspace(): WorkspaceStateV1 {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
        throw new Error("workspace not initialized");
    }
    return JSON.parse(raw) as WorkspaceStateV1;
}

export function saveWorkspace(_ws: WorkspaceStateV1) {
    //window.localStorage.setItem(STORAGE_KEY, JSON.stringify(ws));
}

export function hasWorkspace(): boolean {
    return false; //window.localStorage.getItem(STORAGE_KEY) !== null;
}

export function clearWorkspace() {
    window.localStorage.removeItem(STORAGE_KEY);
}

export function findNode(
    root: WorkspaceDirNode,
    id: WorkspaceNodeId,
): WorkspaceNode | null {
    if (root.id === id) return root;
    const stack: WorkspaceNode[] = [...root.children];
    while (stack.length) {
        const n = stack.pop()!;
        if (n.id === id) return n;
        if (n.type === "dir") {
            for (let i = 0; i < n.children.length; i++) {
                stack.push(n.children[i]);
            }
        }
    }
    return null;
}

export function findParentDirId(
    root: WorkspaceDirNode,
    childId: WorkspaceNodeId,
): WorkspaceNodeId | null {
    if (root.id === childId) return null;
    const stack: WorkspaceDirNode[] = [root];
    while (stack.length) {
        const dir = stack.pop()!;
        for (const c of dir.children) {
            if (c.id === childId) return dir.id;
            if (c.type === "dir") stack.push(c);
        }
    }
    return null;
}

export function isDescendant(
    root: WorkspaceDirNode,
    ancestorId: WorkspaceNodeId,
    targetId: WorkspaceNodeId,
): boolean {
    if (ancestorId === targetId) return true;
    const ancestor = findNode(root, ancestorId);
    if (!ancestor || ancestor.type !== "dir") return false;
    const stack: WorkspaceNode[] = [...ancestor.children];
    while (stack.length) {
        const n = stack.pop()!;
        if (n.id === targetId) return true;
        if (n.type === "dir") {
            for (const c of n.children) stack.push(c);
        }
    }
    return false;
}

export function updateExampleContent(
    root: WorkspaceDirNode,
    id: WorkspaceNodeId,
    content: Pick<
        WorkspaceExampleNode,
        "project" | "inputs" | "activeInputPath"
    >,
): WorkspaceDirNode {
    const visit = (node: WorkspaceNode): WorkspaceNode => {
        if (node.type === "grammar" && node.id === id) {
            return { ...node, ...content };
        }
        if (node.type === "dir") {
            return {
                ...node,
                children: node.children.map(visit),
            };
        }
        return node;
    };
    return visit(root) as WorkspaceDirNode;
}

export function addDir(
    root: WorkspaceDirNode,
    parentId: WorkspaceNodeId,
    dir: WorkspaceDirNode,
): WorkspaceDirNode {
    const visit = (node: WorkspaceNode): WorkspaceNode => {
        if (node.type === "dir" && node.id === parentId) {
            return { ...node, children: [...node.children, dir] };
        }
        if (node.type === "dir") {
            return { ...node, children: node.children.map(visit) };
        }
        return node;
    };
    return visit(root) as WorkspaceDirNode;
}

export function addExample(
    root: WorkspaceDirNode,
    parentId: WorkspaceNodeId,
    ex: WorkspaceExampleNode,
): WorkspaceDirNode {
    const visit = (node: WorkspaceNode): WorkspaceNode => {
        if (node.type === "dir" && node.id === parentId) {
            return { ...node, children: [...node.children, ex] };
        }
        if (node.type === "dir") {
            return { ...node, children: node.children.map(visit) };
        }
        return node;
    };
    return visit(root) as WorkspaceDirNode;
}

export function renameNode(
    root: WorkspaceDirNode,
    id: WorkspaceNodeId,
    name: string,
): WorkspaceDirNode {
    const visit = (node: WorkspaceNode): WorkspaceNode => {
        if (node.id === id) {
            return { ...node, name } as WorkspaceNode;
        }
        if (node.type === "dir") {
            return { ...node, children: node.children.map(visit) };
        }
        return node;
    };
    return visit(root) as WorkspaceDirNode;
}

export function deleteNode(
    root: WorkspaceDirNode,
    id: WorkspaceNodeId,
): WorkspaceDirNode {
    const visitDir = (dir: WorkspaceDirNode): WorkspaceDirNode => {
        return {
            ...dir,
            children: dir.children
                .filter((c) => c.id !== id)
                .map((c) => (c.type === "dir" ? visitDir(c) : c)),
        };
    };
    if (root.id === id) return root;
    return visitDir(root);
}
