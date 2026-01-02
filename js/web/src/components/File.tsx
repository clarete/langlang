import { useCallback, useEffect, useState } from "react";
import {
    FileDirectorySymlinkIcon,
    MortarBoardIcon,
} from "@primer/octicons-react";
import {
    ExpandContent,
    ExpandLabel,
    FileContainer,
    FilePickerButton,
    FilePickerContainer,
    HoverExpandWithIcon,
    PanelHeader,
    PanelHeaderTitle,
} from "./File.styles";
import {
    type LanguageKey,
    type PlaygroundPair,
    playgroundPairs,
    playgroundPairsKeys,
} from "../examples";

interface FileProps {
    title: string;
    selectUrlFromPair: (pair: PlaygroundPair) => UrlFile;
    onContentChange?: (content: string | null) => void;
    children: (
        content: string | null,
        write: (content: string) => void,
    ) => React.ReactNode;
    defaultPair?: PlaygroundPair;
    accept?: Record<string, string[]>;
}

type UrlFile = {
    type: "url";
    id: LanguageKey;
    url: string;
};

type SystemFile = {
    type: "system";
    filename: string;
    handler: FileSystemFileHandle;
};

export type EditorFile = UrlFile | SystemFile;

function ExampleSelect({
    value,
    onChange,
}: {
    value: LanguageKey | undefined;
    onChange: (pair: PlaygroundPair) => void;
}) {
    return (
        <select
            value={value}
            onChange={(e) =>
                onChange(playgroundPairs[e.target.value as LanguageKey])
            }
        >
            {playgroundPairsKeys.map((pair) => (
                <option key={pair} value={pair}>
                    {playgroundPairs[pair].label}
                </option>
            ))}
        </select>
    );
}

function FilePicker({
    value,
    onChange,
    selectUrlFromPair,
    fallbackPair,
    onFilePickerClick,
}: {
    value: EditorFile | null;
    fallbackPair: PlaygroundPair;
    onChange: (editorFile: EditorFile) => void;
    selectUrlFromPair: (pair: PlaygroundPair) => UrlFile;
    onFilePickerClick: () => void;
}) {
    const handleSwitchToUrl = useCallback(() => {
        onChange(selectUrlFromPair(fallbackPair));
    }, [onChange, selectUrlFromPair, fallbackPair]);

    if (value === null || value.type === "url") {
        const id = value?.id ?? fallbackPair.id;
        return (
            <FilePickerContainer>
                <HoverExpandWithIcon expand={value?.type === "url"}>
                    <ExpandContent>
                        <ExampleSelect
                            value={id}
                            onChange={(pair) =>
                                onChange(selectUrlFromPair(pair))
                            }
                        />
                    </ExpandContent>
                    <MortarBoardIcon verticalAlign="middle" />
                </HoverExpandWithIcon>

                <FilePickerButton onClick={onFilePickerClick} title="Open File">
                    <ExpandLabel>Open File</ExpandLabel>
                    <FileDirectorySymlinkIcon verticalAlign="middle" />
                </FilePickerButton>
            </FilePickerContainer>
        );
    } else {
        return (
            <FilePickerContainer>
                <FilePickerButton
                    onClick={handleSwitchToUrl}
                    title="Switch to Examples"
                >
                    <ExpandLabel>Switch to Examples</ExpandLabel>
                    <MortarBoardIcon verticalAlign="middle" />
                </FilePickerButton>
                <FilePickerButton
                    onClick={onFilePickerClick}
                    title="Replace File"
                >
                    <ExpandLabel>Replace File</ExpandLabel>
                    <FileDirectorySymlinkIcon verticalAlign="middle" />
                </FilePickerButton>
            </FilePickerContainer>
        );
    }
}

function File({
    title,
    selectUrlFromPair,
    onContentChange,
    children,
    defaultPair = playgroundPairs.json,
    accept,
}: FileProps) {
    const [editorFile, setEditorFile] = useState<EditorFile | null>(null);
    const [content, setContent] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [status, setStatus] = useState<
        "idle" | "loading" | "success" | "error"
    >("idle");

    const handleContentUpdate = useCallback(
        (newContent: string) => {
            setContent(newContent);
            onContentChange?.(newContent);
        },
        [onContentChange],
    );

    const handleLoadUrlFile = useCallback(
        async ({ url }: UrlFile) => {
            try {
                const response = await fetch(url);
                if (!response.ok) {
                    throw new Error(
                        `Failed to fetch ${url}: ${response.statusText}`,
                    );
                }
                const text = await response.text();
                setContent(text);
                onContentChange?.(text);
                setStatus("success");
                setError(null);
            } catch (e) {
                const msg = e instanceof Error ? e.message : String(e);
                setError(msg);
                setStatus("error");
            }
        },
        [onContentChange],
    );

    const handleUrlChange = useCallback(
        async (editorFile: EditorFile) => {
            setEditorFile(editorFile);
            setStatus("loading");

            if (editorFile.type === "url") {
                handleLoadUrlFile(editorFile);
            } else if (editorFile.type === "system") {
                try {
                    const file = await editorFile.handler.getFile();
                    const text = await file.text();
                    setContent(text);
                    onContentChange?.(text);
                    setStatus("success");
                    setError(null);
                } catch (e) {
                    const msg = e instanceof Error ? e.message : String(e);
                    setError(msg);
                    setStatus("error");
                }
            }
        },
        [onContentChange, handleLoadUrlFile],
    );

    const handleFilePickerClick = useCallback(async () => {
        try {
            if (!window.showOpenFilePicker) {
                alert(
                    "Your browser does not support the File System Access API.",
                );
                return;
            }
            const [handle] = await window.showOpenFilePicker({
                types: accept
                    ? [
                          {
                              description: "Text Files",
                              accept,
                          },
                      ]
                    : undefined,
            });
            if (handle) {
                handleUrlChange({
                    type: "system",
                    filename: handle.name,
                    handler: handle,
                });
            }
        } catch (err) {
            if ((err as any).name !== "AbortError") {
                console.error(err);
            }
        }
    }, [handleUrlChange]);

    useEffect(() => {
        if (editorFile?.type !== "system") return;

        let observer: FileSystemObserver | undefined;

        const startObserving = async () => {
            if ("FileSystemObserver" in window) {
                try {
                    observer = new window.FileSystemObserver(
                        async (records) => {
                            const hasModified = records.some(
                                (r) => r.type === "modified",
                            );
                            if (hasModified) {
                                try {
                                    const file =
                                        await editorFile.handler.getFile();
                                    const text = await file.text();
                                    setContent(text);
                                    onContentChange?.(text);
                                } catch (e) {
                                    console.error(
                                        "Error reading file change:",
                                        e,
                                    );
                                }
                            }
                        },
                    );
                    await observer.observe(editorFile.handler);
                } catch (e) {
                    console.error("Failed to start FileSystemObserver:", e);
                }
            }
        };

        startObserving();

        return () => {
            observer?.disconnect();
        };
    }, [editorFile, onContentChange]);

    useEffect(() => {
        if (editorFile === null) {
            handleUrlChange(selectUrlFromPair(defaultPair));
        }
    }, [editorFile, handleUrlChange, defaultPair, selectUrlFromPair]);

    return (
        <FileContainer>
            <PanelHeader>
                <PanelHeaderTitle>
                    {editorFile?.type === "system"
                        ? editorFile.filename
                        : title}
                    {status === "loading" && " (Loading...)"}
                    {error && ` (Error: ${error})`}
                </PanelHeaderTitle>
                <FilePicker
                    value={editorFile}
                    fallbackPair={defaultPair}
                    onChange={handleUrlChange}
                    selectUrlFromPair={selectUrlFromPair}
                    onFilePickerClick={handleFilePickerClick}
                />
            </PanelHeader>
            {children(content, handleContentUpdate)}
        </FileContainer>
    );
}

export default File;
