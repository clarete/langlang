import { SettingsPanel, SettingsRow } from "./MatcherSettingsPanel.styles";

export type MatcherSettings = {
    captureSpaces: boolean;
    handleSpaces: boolean;
    enableInline: boolean;
    showFails: boolean;
};

type Props = {
    value: MatcherSettings;
    onChange: (next: MatcherSettings) => void;
};

export default function MatcherSettingsPanel({ value, onChange }: Props) {
    return (
        <SettingsPanel>
            <div>
                <SettingsRow title="When off, whitespace-related rules are not wrapped in capture nodes.">
                    <input
                        type="checkbox"
                        checked={value.captureSpaces}
                        onChange={(e) =>
                            onChange({
                                ...value,
                                captureSpaces: e.target.checked,
                            })
                        }
                    />
                    Capture spaces
                </SettingsRow>
                <SettingsRow title="When off, the parser will not show what characters it attempted to match but failed.">
                    <input
                        type="checkbox"
                        checked={value.showFails}
                        onChange={(e) =>
                            onChange({
                                ...value,
                                showFails: e.target.checked,
                            })
                        }
                    />
                    Show fails
                </SettingsRow>
                <SettingsRow title="When off, inlining of definitions is disabled.  Enabling this increases performance at the cost of a larger parser bytecode.">
                    <input
                        type="checkbox"
                        checked={value.enableInline}
                        onChange={(e) =>
                            onChange({
                                ...value,
                                enableInline: e.target.checked,
                            })
                        }
                    />
                    Enable inline
                </SettingsRow>
                <SettingsRow title="When on, langlang injects whitespace handling into the grammar.">
                    <input
                        type="checkbox"
                        checked={value.handleSpaces}
                        onChange={(e) =>
                            onChange({
                                ...value,
                                handleSpaces: e.target.checked,
                            })
                        }
                    />
                    Handle spaces
                </SettingsRow>
            </div>
        </SettingsPanel>
    );
}
