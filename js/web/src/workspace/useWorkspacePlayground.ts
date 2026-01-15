import { useEffect, useMemo, useRef, useState } from "react";
import { defaultProjects } from "./examples";
import type { Project, WorkspaceSelection, WorkspaceStateV1 } from "./types";
import {
    addDir,
    addExample,
    deleteNode,
    findNode,
    findParentDirId,
    isDescendant,
    hasWorkspace,
    clearWorkspace,
    loadWorkspace,
    renameNode,
    saveWorkspace,
    updateExampleContent,
} from "./storage";

function newId(prefix: string) {
    if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
        return `${prefix}:${crypto.randomUUID()}`;
    }
    return `${prefix}:${Date.now().toString(36)}:${Math.random()
        .toString(36)
        .slice(2)}`;
}

export function useWorkspacePlayground() {
    const [workspace, setWorkspace] = useState<WorkspaceStateV1 | null>(null);
    const [selection, setSelection] = useState<WorkspaceSelection | null>(null);
    const [project, setProject] = useState<Project>(() => ({
        entry: "grammar.peg",
        files: [{ path: "grammar.peg", content: "" }],
        activePath: "grammar.peg",
    }));
    const [inputs, setInputs] = useState<
        Array<{ path: string; content: string }>
    >(() => [{ path: "input.txt", content: "" }]);
    const [activeInputPath, setActiveInputPath] = useState("input.txt");
    const pendingGrammarFileRef = useRef<{
        grammarId: string;
        path: string;
    } | null>(null);

    const activeGrammarPath =
        project.activePath ?? project.files[0]?.path ?? project.entry;
    const activeGrammarContent =
        project.files.find((f) => f.path === activeGrammarPath)?.content ?? "";

    const setActiveGrammarPath = (path: string) => {
        setProject((prev) => ({
            ...prev,
            activePath: path,
        }));
    };

    const setActiveGrammarContent = (content: string) => {
        setProject((prev) => ({
            ...prev,
            files: prev.files.map((f) =>
                f.path === activeGrammarPath ? { ...f, content } : f,
            ),
        }));
    };

    const activeInputContent =
        inputs.find((f) => f.path === activeInputPath)?.content ?? "";

    const setActiveInputContent = (content: string) => {
        setInputs((prev) =>
            prev.map((f) =>
                f.path === activeInputPath ? { ...f, content } : f,
            ),
        );
    };

    const selectActiveInputPath = (path: string) => {
        setActiveInputPath(path);
    };

    // Initial load
    useEffect(() => {
        const hydrate = async () => {
            if (hasWorkspace()) {
                try {
                    const ws = loadWorkspace();
                    // If old/stale data contained bad hydration paths, reset.
                    const hasBadPaths =
                        JSON.stringify(ws).includes("undefined");
                    if (hasBadPaths)
                        throw new Error("workspace contains invalid paths");
                    setWorkspace(ws);
                    setSelection(
                        ws.lastSelection ?? { kind: "dir", id: "root" },
                    );
                    return;
                } catch (e) {
                    console.warn("Resetting workspace:", e);
                    clearWorkspace();
                }
            }
            const loaded = await Promise.all(
                defaultProjects.map(async (p) => {
                    const grammarFiles = await Promise.all(
                        p.grammars.map(async (f) => ({
                            path: f.path,
                            content: await fetch(f.url).then((r) => r.text()),
                        })),
                    );
                    const inputFiles = await Promise.all(
                        p.inputs.map(async (f) => ({
                            path: f.path,
                            content: await fetch(f.url).then((r) => r.text()),
                        })),
                    );
                    return {
                        id: p.label
                            .trim()
                            .toLowerCase()
                            .replace(/[^a-z0-9]+/g, "-")
                            .replace(/^-+|-+$/g, ""),
                        label: p.label,
                        grammarFiles,
                        inputFiles,
                    };
                }),
            );
            const ws: WorkspaceStateV1 = {
                version: 1,
                root: {
                    type: "dir",
                    id: "root",
                    name: "Workspace",
                    children: loaded.map((x) => ({
                        type: "grammar",
                        id: `ex:${x.id}`,
                        name: x.label,
                        project: {
                            entry: x.grammarFiles[0]?.path ?? "grammar.peg",
                            files: x.grammarFiles.length
                                ? x.grammarFiles
                                : [{ path: "grammar.peg", content: "" }],
                            activePath:
                                x.grammarFiles[0]?.path ?? "grammar.peg",
                        },
                        inputs: x.inputFiles.length
                            ? x.inputFiles
                            : [{ path: "input.txt", content: "" }],
                        activeInputPath: x.inputFiles[0]?.path ?? "input.txt",
                    })),
                },
                lastSelection: loaded[0]
                    ? { kind: "user", id: `ex:${loaded[0].id}` }
                    : { kind: "dir", id: "root" },
            };

            saveWorkspace(ws);
            setWorkspace(ws);
            setSelection(ws.lastSelection ?? { kind: "dir", id: "root" });
        };

        hydrate().catch((e) => {
            console.error(e);
        });
    }, []);

    // Persist (including last selection)
    useEffect(() => {
        if (!workspace) return;
        saveWorkspace({
            ...workspace,
            ...(selection ? { lastSelection: selection } : {}),
        });
    }, [workspace, selection]);

    // Selection -> load into editors
    useEffect(() => {
        if (!selection) return;

        if (selection.kind === "user") {
            if (!workspace) return;
            const node = findNode(workspace.root, selection.id);
            if (!node)
                throw new Error("workspace selection points to missing node");
            if (node.type !== "grammar") return;
            setProject(node.project);
            setInputs(node.inputs);
            setActiveInputPath(node.activeInputPath);

            const pending = pendingGrammarFileRef.current;
            if (pending && pending.grammarId === node.id) {
                pendingGrammarFileRef.current = null;
                setProject((prev) => ({ ...prev, activePath: pending.path }));
            }
        }
    }, [selection, workspace]);

    useEffect(() => {
        if (!workspace) return;
        if (!selection || selection.kind !== "user") return;

        const node = findNode(workspace.root, selection.id);
        if (node?.type !== "grammar") return;
        const same =
            node.project === project &&
            node.inputs === inputs &&
            node.activeInputPath === activeInputPath;
        if (same) return;

        const handle = window.setTimeout(() => {
            setWorkspace((prev) => {
                if (!prev) return prev;
                const current = findNode(prev.root, selection.id);
                if (current?.type !== "grammar") return prev;
                if (
                    current.project === project &&
                    current.inputs === inputs &&
                    current.activeInputPath === activeInputPath
                )
                    return prev;
                return {
                    ...prev,
                    root: updateExampleContent(prev.root, selection.id, {
                        project,
                        inputs,
                        activeInputPath,
                    }),
                };
            });
        }, 250);
        return () => window.clearTimeout(handle);
    }, [workspace, selection, project, inputs, activeInputPath]);

    const actions = useMemo(() => {
        return {
            createDir(parentId: string, name: string) {
                const trimmed = name.trim();
                if (!trimmed) return null;
                const id = newId("dir");
                setWorkspace((prev) => {
                    if (!prev) return prev;
                    return {
                        ...prev,
                        root: addDir(prev.root, parentId, {
                            type: "dir",
                            id,
                            name: trimmed,
                            children: [],
                        }),
                    };
                });
                return id;
            },
            createExample(parentId: string, name: string) {
                const trimmed = name.trim();
                if (!trimmed) return null;
                const id = newId("grammar");
                const defaultProject: Project = {
                    entry: "grammar.peg",
                    files: [{ path: "grammar.peg", content: "" }],
                    activePath: "grammar.peg",
                };
                setWorkspace((prev) => {
                    if (!prev) return prev;
                    return {
                        ...prev,
                        root: addExample(prev.root, parentId, {
                            type: "grammar",
                            id,
                            name: trimmed,
                            project: defaultProject,
                            inputs: [{ path: "input.txt", content: "" }],
                            activeInputPath: "input.txt",
                        }),
                    };
                });
                return id;
            },
            rename(id: string, name: string) {
                setWorkspace((prev) => {
                    if (!prev) return prev;
                    return { ...prev, root: renameNode(prev.root, id, name) };
                });
            },
            delete(id: string) {
                if (!workspace) return;
                const currentRoot = workspace.root;

                setWorkspace((prev) => {
                    if (!prev) return prev;
                    return { ...prev, root: deleteNode(prev.root, id) };
                });

                setSelection((prevSel) => {
                    if (!prevSel) return { kind: "dir", id: "root" };

                    if (prevSel.kind === "dir" || prevSel.kind === "user") {
                        if (prevSel.id === id)
                            return { kind: "dir", id: "root" };
                        if (isDescendant(currentRoot, id, prevSel.id)) {
                            return { kind: "dir", id: "root" };
                        }
                    }

                    if (prevSel.kind === "user") {
                        const parent =
                            findParentDirId(currentRoot, prevSel.id) ?? "root";
                        if (parent === id) return { kind: "dir", id: "root" };
                    }

                    return prevSel;
                });
            },
            selectGrammarFile(grammarId: string, path: string) {
                // Always record the desired file so it can be applied after a selection-triggered
                // project hydration (e.g. WorkspaceSidebar first calls onSelect, then this).
                pendingGrammarFileRef.current = { grammarId, path };

                // If we're already on the same grammar, update immediately.
                if (selection?.kind === "user" && selection.id === grammarId) {
                    setProject((prev) => ({ ...prev, activePath: path }));
                    return;
                }

                setSelection({ kind: "user", id: grammarId });
            },
        };
    }, [workspace, selection]);

    return {
        workspace,
        selection,
        setSelection,
        project,
        activeGrammarPath,
        setActiveGrammarPath,
        activeGrammarContent,
        setActiveGrammarContent,
        inputs,
        activeInputPath,
        setActiveInputPath: selectActiveInputPath,
        activeInputContent,
        setActiveInputContent,
        ...actions,
    };
}
