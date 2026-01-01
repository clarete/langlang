import { styled } from "@pigment-css/react";

export const PanelHeaderTitle = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    color: "rgba(251, 240, 223, 0.85)",
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.8rem",
    letterSpacing: "0.02em",
    userSelect: "none",
});

export const PanelHeader = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "0.35rem 0.6rem",
    borderBottom: "1px solid rgba(251, 240, 223, 0.2)",
    color: "rgba(251, 240, 223, 0.85)",
});

export const FilePickerContainer = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
});

export const FilePickerButton = styled("button")({
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    borderRadius: '2px',
    background: '#242424',
    marginTop: '1xpx',
    transition: '0.1s',
    cursor: 'var(--bun-cursor)',
    border: 'none',
    '&:hover': {
        background: '#6b6b6b',
        cursor: 'pointer',
    },
});

export const Separator = styled("span")({
    color: '#6b6b6b',
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.8rem",
});
