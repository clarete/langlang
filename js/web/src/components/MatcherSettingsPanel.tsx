import { SettingsPanel, SettingsRow } from "./MatcherSettingsPanel.styles";

export type MatcherSettings = {
    captureSpaces: boolean;
    handleSpaces: boolean;
};

type Props = {
    value: MatcherSettings;
    onChange: (next: MatcherSettings) => void;
};

export default function MatcherSettingsPanel({ value, onChange }: Props) {
    return (
        <SettingsPanel>
            <div>
                <SettingsRow title="When off, whitespace-related rules (Spacing/Space/etc.) are not wrapped in capture nodes.">
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
