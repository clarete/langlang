import { styled } from "@pigment-css/react";

export const Container = styled("div")({
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "0.8rem",
    lineHeight: 1.35,
    padding: "0.5rem",
    userSelect: "none",
});

export const Row = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.35rem",
    padding: "0.1rem 0.25rem",
    borderRadius: 4,
    cursor: "default",
    "&:hover": {
        background: "rgba(255, 255, 255, 0.06)",
    },
});

export const CaretButton = styled("button")({
    width: "1.25rem",
    height: "1.25rem",
    padding: 0,
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    background: "transparent",
    color: "rgba(251, 240, 223, 0.9)",
    border: "none",
    cursor: "pointer",
    borderRadius: 4,
    "&:hover": {
        background: "rgba(255, 255, 255, 0.08)",
    },
});

export const PlaceholderCaret = styled("div")({
    width: "1.25rem",
    height: "1.25rem",
});

export const Label = styled("span")({
    color: "rgba(251, 240, 223, 0.92)",
    whiteSpace: "nowrap",
});

export const Meta = styled("span")({
    color: "rgba(251, 240, 223, 0.55)",
    marginLeft: "0.35rem",
    whiteSpace: "nowrap",
});


