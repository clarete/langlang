export type WorkspaceNodeId = string;

export type WorkspaceNode = WorkspaceDirNode | WorkspaceExampleNode;

export interface WorkspaceDirNode {
    type: "dir";
    id: WorkspaceNodeId;
    name: string;
    children: WorkspaceNode[];
}

export interface WorkspaceExampleNode {
    type: "grammar";
    id: WorkspaceNodeId;
    name: string;
    project: Project;
    inputs: InputFile[];
    activeInputPath: string;
}

export interface ProjectFile {
    path: string;
    content: string;
}

export interface InputFile {
    path: string;
    content: string;
}

export interface Project {
    entry: string;
    files: ProjectFile[];
    activePath?: string;
}

export interface WorkspaceStateV1 {
    version: 1;
    root: WorkspaceDirNode;
    lastSelection?: WorkspaceSelection;
}

export type WorkspaceSelection =
    | { kind: "user"; id: WorkspaceNodeId }
    | { kind: "dir"; id: WorkspaceNodeId };
