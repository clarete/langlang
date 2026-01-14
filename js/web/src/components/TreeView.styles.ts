import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const Container = styled("div")({
    fontFamily: theme.fonts.mono,
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
        background: theme.colors.hover.row,
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
    color: theme.colors.text.strong,
    border: "none",
    cursor: "pointer",
    borderRadius: 4,
    "&:hover": {
        background: theme.colors.hover.control,
    },
});

export const PlaceholderCaret = styled("div")({
    width: "1.25rem",
    height: "1.25rem",
});

export const Label = styled("span")({
    color: theme.colors.text.stronger,
    whiteSpace: "nowrap",
});

export const Meta = styled("span")({
    color: theme.colors.text.muted,
    marginLeft: "0.35rem",
    whiteSpace: "nowrap",
});
