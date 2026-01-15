import { useEffect, useMemo, useState } from "react";
import {
    FileDirectorySymlinkIcon,
    FileDirectoryIcon,
    FileIcon,
    PlusIcon,
    PencilIcon,
    TrashIcon,
} from "@primer/octicons-react";
import type {
    WorkspaceDirNode,
    WorkspaceNode,
    WorkspaceNodeId,
    WorkspaceSelection,
    WorkspaceStateV1,
} from "../workspace/types";
import { findNode, findParentDirId } from "../workspace/storage";
import {
    ActionButton,
    CaretButton,
    Label,
    PlaceholderCaret,
    Row,
    RowActionButton,
    RowActions,
    Section,
    SidebarBody,
    SidebarHeader,
    SidebarHeaderActions,
    SidebarHeaderTitle,
    SidebarRoot,
} from "./WorkspaceSidebar.styles";

function isSelected(
    sel: WorkspaceSelection | null,
    kind: WorkspaceSelection["kind"],
    id: string,
) {
    return sel?.kind === kind && sel.id === id;
}

function listAllDirIds(root: WorkspaceDirNode): Set<WorkspaceNodeId> {
    const out = new Set<WorkspaceNodeId>();
    const stack: WorkspaceNode[] = [root];
    while (stack.length) {
        const n = stack.pop()!;
        if (n.type === "dir") {
            out.add(n.id);
            for (let i = 0; i < n.children.length; i++)
                stack.push(n.children[i]);
        }
    }
    return out;
}

function listAllExpandableIds(root: WorkspaceDirNode): Set<WorkspaceNodeId> {
    const out = new Set<WorkspaceNodeId>();
    const stack: WorkspaceNode[] = [root];
    while (stack.length) {
        const n = stack.pop()!;
        if (n.type === "dir" || n.type === "grammar") {
            out.add(n.id);
        }
        if (n.type === "dir") {
            for (let i = 0; i < n.children.length; i++)
                stack.push(n.children[i]);
        }
    }
    return out;
}

export interface WorkspaceSidebarProps {
    workspace: WorkspaceStateV1;
    selection: WorkspaceSelection | null;
    onSelect: (sel: WorkspaceSelection) => void;
    onSelectGrammarFile: (grammarId: WorkspaceNodeId, path: string) => void;
    onCreateDir: (
        parentId: WorkspaceNodeId,
        name: string,
    ) => WorkspaceNodeId | null;
    onCreateExample: (
        parentId: WorkspaceNodeId,
        name: string,
    ) => WorkspaceNodeId | null;
    onRenameNode: (id: WorkspaceNodeId, name: string) => void;
    onDeleteNode: (id: WorkspaceNodeId) => void;
}

export default function WorkspaceSidebar({
    workspace,
    selection,
    onSelect,
    onSelectGrammarFile,
    onCreateDir,
    onCreateExample,
    onRenameNode,
    onDeleteNode,
}: WorkspaceSidebarProps) {
    const defaultExpanded = useMemo(
        () => listAllDirIds(workspace.root),
        [workspace.root],
    );
    const allExpandableIds = useMemo(
        () => listAllExpandableIds(workspace.root),
        [workspace.root],
    );
    const [expanded, setExpanded] = useState<Set<WorkspaceNodeId>>(
        () => defaultExpanded,
    );
    useEffect(() => {
        setExpanded((prev) => {
            const next = new Set<WorkspaceNodeId>();
            for (const id of prev) {
                // Keep anything that still exists (dirs and "project" nodes).
                if (allExpandableIds.has(id)) next.add(id);
            }
            // Always keep root visible/expandable
            next.add("root");
            return next.size ? next : defaultExpanded;
        });
    }, [defaultExpanded, allExpandableIds]);

    const toggleDir = (id: WorkspaceNodeId) => {
        setExpanded((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    };

    const handleCreateDir = () => {
        const name = window.prompt("Folder name?");
        if (!name) return;
        const parentId = getDefaultParentDirId();
        const newId = onCreateDir(parentId, name.trim());
        if (!newId) return;
        setExpanded((prev) => new Set(prev).add(parentId).add(newId));
        onSelect({ kind: "dir", id: newId });
    };

    const handleCreateGrammar = () => {
        const name = window.prompt("Grammar name?");
        if (!name) return;
        const parentId = getDefaultParentDirId();
        const newId = onCreateExample(parentId, name.trim());
        if (!newId) return;
        setExpanded((prev) => new Set(prev).add(parentId));
        onSelect({ kind: "user", id: newId });
    };

    const getDefaultParentDirId = () => {
        if (selection?.kind === "dir") return selection.id;
        if (selection?.kind === "user") {
            return findParentDirId(workspace.root, selection.id) ?? "root";
        }
        return "root";
    };

    const handleRename = (nodeId: WorkspaceNodeId, currentName: string) => {
        const next = window.prompt("Rename to:", currentName);
        if (!next) return;
        const trimmed = next.trim();
        if (!trimmed) return;
        onRenameNode(nodeId, trimmed);
    };

    const handleDelete = (nodeId: WorkspaceNodeId) => {
        if (nodeId === "root") return;
        const node = findNode(workspace.root, nodeId);
        if (!node) return;
        const ok =
            node.type === "dir"
                ? window.confirm(
                      `Delete folder "${node.name}" and all its contents? This cannot be undone.`,
                  )
                : window.confirm(
                      `Delete grammar "${node.name}"? This cannot be undone.`,
                  );
        if (!ok) return;
        onDeleteNode(nodeId);
    };

    const canDeleteSelection =
        selection?.kind === "user" ||
        (selection?.kind === "dir" && selection.id !== "root");

    const handleDeleteSelection = () => {
        if (!selection) return;
        if (selection.kind === "user") handleDelete(selection.id);
        if (selection.kind === "dir") handleDelete(selection.id);
    };

    const renderWorkspaceNode = (node: WorkspaceNode, depth: number) => {
        if (node.type === "dir") {
            const isExpanded = expanded.has(node.id);
            const caret = (
                <CaretButton
                    type="button"
                    aria-label={
                        isExpanded ? "Collapse folder" : "Expand folder"
                    }
                    onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        toggleDir(node.id);
                    }}
                >
                    {isExpanded ? "▾" : "▸"}
                </CaretButton>
            );
            return (
                <div key={node.id}>
                    <Row
                        selected={isSelected(selection, "dir", node.id)}
                        onClick={() => onSelect({ kind: "dir", id: node.id })}
                        role="button"
                        tabIndex={0}
                        style={{ paddingLeft: `${0.35 + depth * 0.85}rem` }}
                    >
                        {caret}
                        <FileDirectoryIcon verticalAlign="middle" />
                        <Label title={node.name}>{node.name}</Label>
                        {node.id !== "root" ? (
                            <RowActions
                                style={{
                                    opacity: isSelected(
                                        selection,
                                        "dir",
                                        node.id,
                                    )
                                        ? 1
                                        : undefined,
                                }}
                            >
                                <RowActionButton
                                    type="button"
                                    title="Rename folder"
                                    onClick={(e) => {
                                        e.preventDefault();
                                        e.stopPropagation();
                                        handleRename(node.id, node.name);
                                    }}
                                >
                                    <PencilIcon verticalAlign="middle" />
                                </RowActionButton>
                                <RowActionButton
                                    type="button"
                                    title="Delete folder"
                                    onClick={(e) => {
                                        e.preventDefault();
                                        e.stopPropagation();
                                        handleDelete(node.id);
                                    }}
                                >
                                    <TrashIcon verticalAlign="middle" />
                                </RowActionButton>
                            </RowActions>
                        ) : null}
                    </Row>
                    {isExpanded
                        ? node.children.map((c) =>
                              renderWorkspaceNode(c, depth + 1),
                          )
                        : null}
                </div>
            );
        }

        // Treat each grammar node as a "project directory" that expands to show its grammar files.
        const isExpanded = expanded.has(node.id);
        const caret = (
            <CaretButton
                type="button"
                aria-label={isExpanded ? "Collapse project" : "Expand project"}
                onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    toggleDir(node.id);
                }}
            >
                {isExpanded ? "▾" : "▸"}
            </CaretButton>
        );

        return (
            <div key={node.id}>
                <Row
                    selected={isSelected(selection, "user", node.id)}
                    onClick={() => onSelect({ kind: "user", id: node.id })}
                    role="button"
                    tabIndex={0}
                    style={{ paddingLeft: `${0.35 + depth * 0.85}rem` }}
                >
                    {caret}
                    <FileDirectoryIcon verticalAlign="middle" />
                    <Label title={node.name}>{node.name}</Label>
                    <RowActions
                        style={{
                            opacity: isSelected(selection, "user", node.id)
                                ? 1
                                : undefined,
                        }}
                    >
                        <RowActionButton
                            type="button"
                            title="Rename grammar"
                            onClick={(e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                handleRename(node.id, node.name);
                            }}
                        >
                            <PencilIcon verticalAlign="middle" />
                        </RowActionButton>
                        <RowActionButton
                            type="button"
                            title="Delete grammar"
                            onClick={(e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                handleDelete(node.id);
                            }}
                        >
                            <TrashIcon verticalAlign="middle" />
                        </RowActionButton>
                    </RowActions>
                </Row>
                {isExpanded
                    ? node.project.files.map((f) => (
                          <Row
                              key={`${node.id}:${f.path}`}
                              selected={
                                  isSelected(selection, "user", node.id) &&
                                  node.project.activePath === f.path
                              }
                              onClick={() => {
                                  // Keep this "project folder" expanded when selecting a file.
                                  setExpanded((prev) =>
                                      new Set(prev).add(node.id),
                                  );
                                  onSelect({ kind: "user", id: node.id });
                                  onSelectGrammarFile(node.id, f.path);
                              }}
                              role="button"
                              tabIndex={0}
                              style={{
                                  paddingLeft: `${0.35 + (depth + 1) * 0.85}rem`,
                              }}
                          >
                              <PlaceholderCaret />
                              <FileIcon verticalAlign="middle" />
                              <Label title={f.path}>{f.path}</Label>
                          </Row>
                      ))
                    : null}
            </div>
        );
    };

    return (
        <SidebarRoot>
            <SidebarHeader>
                <SidebarHeaderTitle>Workspace</SidebarHeaderTitle>
                <SidebarHeaderActions>
                    <ActionButton
                        type="button"
                        title="New folder"
                        onClick={handleCreateDir}
                    >
                        <FileDirectorySymlinkIcon verticalAlign="middle" />
                    </ActionButton>
                    <ActionButton
                        type="button"
                        title="New grammar"
                        onClick={handleCreateGrammar}
                    >
                        <PlusIcon verticalAlign="middle" />
                    </ActionButton>
                    <ActionButton
                        type="button"
                        title="Delete selected"
                        onClick={handleDeleteSelection}
                        disabled={!canDeleteSelection}
                    >
                        <TrashIcon verticalAlign="middle" />
                    </ActionButton>
                </SidebarHeaderActions>
            </SidebarHeader>
            <SidebarBody>
                <Section>{renderWorkspaceNode(workspace.root, 0)}</Section>
            </SidebarBody>
        </SidebarRoot>
    );
}
