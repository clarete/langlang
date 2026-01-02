import { styled } from "@pigment-css/react";

export const SettingsPanel = styled("div")({
    padding: "0.6rem 0.75rem",
    background: "#242424",
    border: "1px solid rgba(251, 240, 223, 0.18)",
    borderTop: "none",
    marginTop: "-1px",
});

export const SettingsRow = styled("label")({
    display: "flex",
    alignItems: "center",
    gap: "0.55rem",
    padding: "0.35rem 0",
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.78rem",
    color: "rgba(251, 240, 223, 0.85)",
    userSelect: "none",
});
