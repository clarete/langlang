import { styled } from "@pigment-css/react";

export const FileContainer = styled("div")({
    display: "flex",
    flexDirection: "column",
    height: "100%",
});

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

export const FilePickerButton = styled("button")<{
    expanded?: boolean;
    color?: string;
}>({
    display: "grid",
    gridTemplateColumns: "0fr 1.25rem",
    gap: "0rem",
    alignItems: "center",
    justifyContent: "center",
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    borderRadius: "2px",
    background: (props) => props.color ?? "#242424",
    marginTop: "1xpx",
    fontSize: "0.8rem",
    transition: "150ms",
    cursor: "var(--bun-cursor)",
    border: "none",

    "& svg": {
        width: "0.8rem",
        height: "0.8rem",
        padding: "0.15rem",
    },

    "&:hover": {
        background: "#6b6b6b",
        cursor: "pointer",
        gridTemplateColumns: "1fr 1.25rem",
        gap: "0.5rem",
    },

    variants: [
        {
            props: { expanded: true },
            style: {
                gridTemplateColumns: "1fr 1.25rem",
                gap: "0.5rem",
            },
        },
    ],
});

export const HoverExpandWithIcon = styled("div")<{ expand: boolean }>({
    display: "grid",
    gridTemplateColumns: "0fr 1.25rem",
    gap: "0rem",
    alignItems: "center",
    justifyContent: "center",
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    borderRadius: "2px",
    fontSize: "0.8rem",
    background: "#242424",
    transition: "150ms",
    cursor: "var(--bun-cursor)",
    border: "none",
    color: "buttontext",

    "& svg": {
        width: "0.8rem",
        height: "0.8rem",
        padding: "0.15rem",
    },

    variants: [
        {
            props: { expand: true },
            style: {
                gridTemplateColumns: "1fr 1.25rem",
                gap: "0.5rem",
            },
        },
        {
            props: { expand: false },
            style: {
                "&:hover": {
                    cursor: "pointer",
                    gridTemplateColumns: "1fr 1.25rem",
                    gap: "0.5rem",
                },
            },
        },
    ],
});

export const ExpandLabel = styled("span")({
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.50rem",
    whiteSpace: "nowrap",
    overflow: "hidden",
});

export const ExpandContent = styled("div")({
    fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.50rem",
    whiteSpace: "nowrap",
    overflow: "hidden",
});
