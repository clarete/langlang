import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const SidebarRoot = styled("div")({
    height: "100%",
    display: "flex",
    flexDirection: "column",
    minWidth: 0,
    maxWidth: "100%",
    background: theme.colors.bg.sidebar,
    borderRight: `1px solid ${theme.colors.border.subtle}`,
});

export const SidebarHeader = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "0.35rem 0.6rem",
    borderBottom: `1px solid ${theme.colors.border.subtle}`,
    color: theme.colors.text.primary,
    fontFamily: theme.fonts.mono,
    fontSize: "0.78rem",
    letterSpacing: "0.02em",
    userSelect: "none",
});

export const SidebarHeaderTitle = styled("div")({
    fontWeight: 700,
});

export const SidebarHeaderActions = styled("div")({
    display: "flex",
    gap: "0.35rem",
});

export const ActionButton = styled("button")({
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: "1.55rem",
    height: "1.55rem",
    borderRadius: theme.radii.md,
    border: `1px solid ${theme.colors.border.subtle2}`,
    background: "rgba(255, 255, 255, 0.04)",
    color: theme.colors.text.primary,
    cursor: "pointer",
    transition: "0.12s",
    "&:hover": {
        background: theme.colors.hover.control,
        borderColor: theme.colors.border.hover,
    },
    "& svg": {
        width: "0.9rem",
        height: "0.9rem",
    },
    "&:disabled": {
        opacity: 0.45,
        cursor: "not-allowed",
    },
    "&:disabled:hover": {
        background: "rgba(255, 255, 255, 0.04)",
        borderColor: theme.colors.border.subtle2,
    },
});

export const SidebarBody = styled("div")({
    flex: 1,
    minHeight: 0,
    overflow: "auto",
    maxWidth: "100%",
    // overflowX: "hidden",
    // padding: "0.35rem 0.25rem",
});

export const Section = styled("div")({
    maxWidth: "100%",
    padding: "0.35rem 0.25rem",
});

export const SectionTitle = styled("div")({
    padding: "0.15rem 0.35rem",
    marginBottom: "0.15rem",
    color: theme.colors.text.muted,
    fontFamily: theme.fonts.mono,
    fontSize: "0.72rem",
    textTransform: "uppercase",
    letterSpacing: "0.06em",
    userSelect: "none",
});

export const Row = styled("div")<{ selected?: boolean }>({
    display: "flex",
    alignItems: "center",
    gap: "0.35rem",
    padding: "0.18rem 0.35rem",
    borderRadius: theme.radii.md,
    cursor: "pointer",
    background: "transparent",
    border: "none",
    color: theme.colors.text.primary,
    fontFamily: theme.fonts.mono,
    fontSize: "0.8rem",
    textAlign: "left",
    userSelect: "none",
    minWidth: 0,

    "&:hover": {
        background: theme.colors.hover.row,
    },

    variants: [
        {
            props: { selected: true },
            style: {
                background: theme.colors.border.subtle,
                color: theme.colors.text.strongest,
            },
        },
    ],
});

export const RowActions = styled("div")({
    display: "flex",
    gap: "0.15rem",
    opacity: 0,
    transition: "0.12s",
    flex: "0 0 auto",
    [`${Row}:hover &`]: {
        opacity: 1,
    },
});

export const RowActionButton = styled("button")({
    width: "1.35rem",
    height: "1.35rem",
    padding: 0,
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    borderRadius: theme.radii.md,
    border: `1px solid ${theme.colors.border.subtle}`,
    background: "rgba(0, 0, 0, 0.12)",
    color: theme.colors.text.primary,
    cursor: "pointer",
    "&:hover": {
        background: theme.colors.hover.control,
        borderColor: theme.colors.border.hover2,
    },
    "& svg": {
        width: "0.85rem",
        height: "0.85rem",
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
    borderRadius: theme.radii.md,
    "&:hover": {
        background: theme.colors.hover.control,
    },
});

export const PlaceholderCaret = styled("div")({
    width: "1.25rem",
    height: "1.25rem",
});

export const Label = styled("span")({
    whiteSpace: "nowrap",
    overflow: "hidden",
    textOverflow: "ellipsis",
    flex: 1,
    minWidth: 0,
});
