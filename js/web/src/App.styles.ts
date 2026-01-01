import { styled } from "@pigment-css/react";

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

export const EndpointRow = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
    background: "#1a1a1a",
    padding: "0.75rem",
    borderRadius: "12px",
    fontFamily: "monospace",
    border: "2px solid #fbf0df",
    transition: "0.3s",
    width: "100%",
    boxSizing: "border-box",
    "&:focus-within": {
        borderColor: "#f3d5a3",
    },
});

export const MethodSelect = styled("select")({
    background: "#fbf0df",
    color: "#1a1a1a",
    padding: "0.3rem 0.7rem",
    borderRadius: "8px",
    fontWeight: 700,
    fontSize: "0.9em",
    appearance: "none",
    margin: 0,
    width: "min-content",
    display: "block",
    flexShrink: 0,
    border: "none",
    "& option": {
        textAlign: "left",
    },
});

export const UrlInput = styled("input")({
    width: "100%",
    flex: 1,
    background: 0,
    border: 0,
    color: "#fbf0df",
    font: "1em monospace",
    padding: "0.2rem",
    outline: 0,
    "&:focus": {
        color: "#fff",
    },
    "&::placeholder": {
        color: "rgba(251, 240, 223, 0.4)",
    },
});

export const ResponseArea = styled("textarea")({
    width: "100%",
    minHeight: "120px",
    background: "#1a1a1a",
    padding: "0.75rem",
    color: "#fbf0df",
    fontFamily: "monospace",
    resize: "vertical",
    boxSizing: "border-box",

    "&:focus": {
        borderColor: "#f3d5a3",
        outline: "none",
    },
    "&::placeholder": {
        color: "rgba(251, 240, 223, 0.4)",
    },
});

export const PlaygroundContainer = styled("div")({
    width: "100%",
    display: "grid",
    gridTemplateColumns: "auto 1fr",
    gridTemplateRows: "repeat(2, minmax(0, 15rem))",
    gap: "1rem",
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
    height: "100%",
    minHeight: 0,
});

export const OutputTabs = styled("div")({
    display: "flex",
    flexDirection: "column",
    borderLeft: "1px solid rgba(251, 240, 223, 0.2)",
    background: "rgba(0, 0, 0, 0.12)",
});

export const OutputTab = styled("button")<{ active?: boolean }>({
    width: "2.4rem",
    flex: "0 0 auto",
    padding: "0.5rem 0.25rem",
    border: "none",
    borderBottom: "1px solid rgba(251, 240, 223, 0.15)",
    background: "transparent",
    color: "rgba(251, 240, 223, 0.75)",
    cursor: "pointer",
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.75rem",
    letterSpacing: "0.04em",
    writingMode: "vertical-rl",
    transform: "rotate(180deg)",
    userSelect: "none",
    variants: [
        {
            props: { active: true },
            style: {
                background: "rgba(251, 240, 223, 0.12)",
                color: "rgba(251, 240, 223, 0.95)",
            },
        },
    ],
});

export const ResponseContainer = styled("div")({
    flex: 1,
});

export const PanelContainer = styled("div")({
    background: "#1a1a1a",
    height: "100%",
    width: "100%",
    display: "flex",
    flexDirection: "column",
});

export const PanelHeader = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "0.35rem 0.6rem",
    borderBottom: "1px solid rgba(251, 240, 223, 0.2)",
    color: "rgba(251, 240, 223, 0.85)",
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.8rem",
    letterSpacing: "0.02em",
    userSelect: "none",
});

export const PanelBody = styled("div")({
    flex: 1,
    minHeight: 0,
});
