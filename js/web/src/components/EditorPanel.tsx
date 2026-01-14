import type { EditorProps, Monaco } from "@monaco-editor/react";
import { Editor } from "@monaco-editor/react";
import { PanelBody, PanelContainer, PanelHeader } from "./Panel.styles";

export interface EditorPanelProps {
    title: string;
    language: string;
    value: string;
    options: EditorProps["options"];
    theme?: string;
    beforeMount?: EditorProps["beforeMount"];
    onMount?: (editor: unknown, monaco: Monaco) => void;
    onChange: (next: string) => void;
    headerRight?: React.ReactNode;
}

export default function EditorPanel({
    title,
    language,
    value,
    options,
    theme = "vs-dark",
    beforeMount,
    onMount,
    onChange,
    headerRight,
}: EditorPanelProps) {
    return (
        <PanelContainer>
            <PanelHeader>
                <div style={{ minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {title}
                </div>
                {headerRight ? <div>{headerRight}</div> : null}
            </PanelHeader>
            <PanelBody>
                <Editor
                    theme={theme}
                    beforeMount={beforeMount}
                    language={language}
                    height="100%"
                    width="100%"
                    options={options}
                    value={value}
                    onMount={onMount as any}
                    onChange={(v) => onChange(v ?? "")}
                />
            </PanelBody>
        </PanelContainer>
    );
}
