import { styled } from "@pigment-css/react";
import { theme } from "./theme";

export const BarRoot = styled("div")({
    background: theme.colors.bg.bar,
    borderTop: `1px solid ${theme.colors.border.normal}`,
    flexShrink: 0,
});

export const BarHeader = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
    padding: "0.5rem 0.75rem",
    color: theme.colors.text.strong,
    fontFamily: theme.fonts.mono,
    fontSize: "0.82rem",
    userSelect: "none",
    borderBottom: `1px solid ${theme.colors.border.normal}`,
});

export const Status = styled("div")({
    fontSize: "0.8rem",
});

export const SettingsTab = styled("button")<{ active?: boolean }>({
    fontFamily: theme.fonts.mono,
    borderRadius: theme.radii.lg,
    background: theme.colors.bg.control,
    color: theme.colors.text.strong,
    padding: "0.28rem 0.6rem",
    border: `1px solid ${theme.colors.border.normal}`,
    transition: "0.12s",
    cursor: "pointer",
    position: "relative",
    marginBottom: "-1px",
    "&:hover": {
        background: theme.colors.bg.controlHover,
        borderColor: theme.colors.border.active2,
    },
    variants: [
        {
            props: { active: true },
            style: {
                background: theme.colors.bg.control,
                borderColor: theme.colors.border.active,
                borderBottomColor: theme.colors.bg.control,
                borderRadius: "8px 8px 0 0",
            },
        },
    ],
});

export const OutputViewContainerWrapper = styled("div")({
    minWidth: "300px",
    flex: 1,
    minHeight: 0,
    height: "100%",
    overflow: "auto",
});

export const OutputPanelBody = styled("div")({
    display: "flex",
    flexDirection: "column",
    height: "100%",
    minHeight: 0,
});

export const OutputView = styled("div")({
    flex: 1,
    minHeight: 0,
    display: "flex",
});

export const ErrorDisplay = styled("div")({
    padding: "0.75rem",
    color: "rgba(255, 123, 123, 0.9)",
    fontFamily: theme.fonts.mono,
    whiteSpace: "pre-wrap",
});
