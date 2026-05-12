import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const Tabs = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.25rem",
    minWidth: 0,
    overflowX: "auto",
    overflowY: "hidden",
    flex: 1,
});

export const Tab = styled("button")<{ active?: boolean }>({
    background: "transparent",
    color: theme.colors.text.primary,
    padding: "0.25rem 0.6rem",
    cursor: "pointer",
    fontFamily: theme.fonts.mono,
    fontSize: "0.72rem",
    letterSpacing: "0.04em",
    maxWidth: "16rem",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
    border: 0,
    "&:hover": {
        background: theme.colors.hover.row,
        borderColor: theme.colors.border.hover,
    },
    variants: [
        {
            props: { active: true },
            style: {
                background: theme.colors.bg.editor,
                borderColor: theme.colors.border.active,
                borderBottomColor: theme.colors.bg.editor,
                color: theme.colors.text.strongest,
            },
        },
    ],
});
