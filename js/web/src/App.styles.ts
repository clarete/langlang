import { styled } from "@pigment-css/react";
import { theme } from "./theme";

export const Main = styled("div")({
    margin: "0 auto",
    textAlign: "center",
    position: "relative",
    zIndex: 1,
    display: "flex",
    flexDirection: "column",
    width: "100%",
    minHeight: "calc(100vh - 2rem)",
    padding: "1rem 0",
});

export const LogoContainer = styled("div")({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    gap: "2rem",
    marginBottom: "2rem",
});

export const Logo = styled("img")({
    height: "6em",
    padding: "1.5em",
    willChange: "filter",
    transition: "filter 0.3s",
    "&:hover": {
        filter: "drop-shadow(0 0 2em #646cffaa)",
    },
    variants: [
        {
            props: { className: "bun-logo" },
            style: {
                transform: "scale(1.2)",
                "&:hover": {
                    filter: "drop-shadow(0 0 2em #fbf0dfaa)",
                },
            },
        },
        {
            props: { className: "react-logo" },
            style: {
                animation: "spin 20s linear infinite",
                "&:hover": {
                    filter: "drop-shadow(0 0 2em #61dafbaa)",
                },
            },
        },
    ],
});

export const ApiTester = styled("div")({
    margin: "2rem auto 0",
    width: "100%",
    maxWidth: "600px",
    textAlign: "left",
    display: "flex",
    flexDirection: "column",
    gap: "1rem",
});

export const BarRoot = styled("div")({
    position: "sticky",
    top: 0,
    zIndex: 10,
    background: theme.colors.bg.bar,
    borderBottom: `1px solid ${theme.colors.border.normal}`,
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

export const BarTitle = styled("div")({
    fontWeight: 700,
    letterSpacing: "0.02em",
});

export const BarSpacer = styled("div")({
    flex: 1,
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
