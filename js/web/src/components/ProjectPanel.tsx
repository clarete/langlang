import { Editor, type EditorProps } from "@monaco-editor/react";
import type { Project } from "../workspace/types";
import { PanelBody, PanelContainer, PanelHeader } from "./Panel.styles";
import { HeaderInner } from "./ProjectPanel.styles";
import { Tab, Tabs } from "./Tabs.styles";
import { toDocUri } from "../monaco/lsp";
import { PEG_THEME_ID } from "../monaco/peg";

export interface ProjectPanelProps {
    project: Project;
    activePath: string;
    onActivePathChange: (path: string) => void;
    value: string;
    onChange: (next: string) => void;
    options: EditorProps["options"];
    beforeMount?: EditorProps["beforeMount"];
    onMount?: EditorProps["onMount"];
    theme?: string;
}

function basename(path: string) {
    const ix = path.lastIndexOf("/");
    return ix >= 0 ? path.slice(ix + 1) : path;
}

export default function ProjectPanel({
    project,
    activePath,
    onActivePathChange,
    value,
    onChange,
    options,
    beforeMount,
    onMount,
    theme = PEG_THEME_ID,
}: ProjectPanelProps) {
    return (
        <PanelContainer>
            <PanelHeader>
                <HeaderInner>
                    <Tabs>
                        {project.files.map((f) => (
                            <Tab
                                key={f.path}
                                type="button"
                                active={f.path === activePath}
                                onClick={() => onActivePathChange(f.path)}
                                title={f.path}
                            >
                                {basename(f.path)}
                            </Tab>
                        ))}
                    </Tabs>
                </HeaderInner>
            </PanelHeader>
            <PanelBody>
                <Editor
                    theme={theme}
                    beforeMount={beforeMount}
                    language="peg"
                    path={toDocUri(activePath)}
                    height="100%"
                    width="100%"
                    options={options}
                    value={value}
                    onMount={onMount}
                    onChange={(v) => onChange(v ?? "")}
                />
            </PanelBody>
        </PanelContainer>
    );
}
