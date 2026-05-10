import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const LiveEditorRoot = styled("div")({
    display: "flex",
    flexDirection: "column",
    background: theme.colors.bg.panel,
    border: `1px solid ${theme.colors.border.normal}`,
    borderRadius: theme.radii.lg,
    overflow: "hidden",
    fontFamily: theme.fonts.mono,
});

export const LiveEditorBody = styled("div")({
    flex: 1,
    minHeight: 0,
    display: "flex",
    overflow: "hidden",
});

export const LiveEditorStatusBar = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "flex-end",
    gap: "0.5rem",
    padding: "0.2rem 0.6rem",
    borderTop: `1px solid ${theme.colors.border.subtle}`,
    background: theme.colors.bg.bar,
    fontSize: "0.72rem",
    color: theme.colors.text.muted,
    userSelect: "none",
});

export const LiveEditorStatusDot = styled("span")<{ error?: boolean }>({
    display: "inline-block",
    width: "6px",
    height: "6px",
    borderRadius: "50%",
    background: theme.colors.accent.cream,
    variants: [
        {
            props: { error: true },
            style: { background: "rgba(255, 123, 123, 0.9)" },
        },
    ],
});

export const LiveEditorSide = styled("div")({
    display: "flex",
    flexDirection: "column",
    height: "100%",
    overflow: "hidden",
});

export const LiveEditorInputArea = styled("textarea")({
    flex: 1,
    resize: "none",
    background: theme.colors.bg.editor,
    color: theme.colors.text.primary,
    border: "none",
    outline: "none",
    fontFamily: theme.fonts.mono,
    fontSize: "13px",
    lineHeight: 1.5,
    padding: "0.5rem",
    width: "100%",
    boxSizing: "border-box",
    "&::placeholder": {
        color: theme.colors.text.placeholder,
    },
});

export const LiveEditorPanelLabel = styled("div")({
    padding: "0.2rem 0.5rem",
    fontSize: "0.72rem",
    color: theme.colors.text.muted,
    borderBottom: `1px solid ${theme.colors.border.subtle}`,
    background: theme.colors.bg.bar,
    userSelect: "none",
    letterSpacing: "0.04em",
    textTransform: "uppercase",
});

export const LiveEditorLoading = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    height: "100%",
    color: theme.colors.text.muted,
    fontSize: "0.82rem",
    fontFamily: theme.fonts.mono,
});

export const LiveEditorOutputPane = styled("div")({
    height: "100%",
    overflow: "auto",
    display: "flex",
    flexDirection: "column",
});

export const LiveEditorTabs = styled("div")({
    display: "flex",
    gap: "0.1rem",
    padding: "0.2rem 0.4rem 0",
    background: theme.colors.bg.bar,
    borderBottom: `1px solid ${theme.colors.border.subtle}`,
});

export const LiveEditorTab = styled("button")<{ active?: boolean }>({
    fontFamily: theme.fonts.mono,
    fontSize: "0.72rem",
    padding: "0.15rem 0.5rem",
    background: "transparent",
    border: `1px solid transparent`,
    borderBottom: "none",
    color: theme.colors.text.muted,
    cursor: "pointer",
    borderRadius: "4px 4px 0 0",
    letterSpacing: "0.03em",
    variants: [
        {
            props: { active: true },
            style: {
                color: theme.colors.text.strong,
                background: theme.colors.bg.panel,
                borderColor: theme.colors.border.subtle,
            },
        },
    ],
});
