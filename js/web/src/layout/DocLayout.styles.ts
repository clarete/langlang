import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const DocRoot = styled("div")({
    minHeight: "100vh",
    background: theme.colors.bg.app,
    color: theme.colors.text.primary,
    display: "flex",
    flexDirection: "column",
});

export const DocMain = styled("main")({
    flex: 1,
    width: "100%",
    maxWidth: "72rem",
    margin: "0 auto",
    padding: "2rem 1.5rem",
    display: "flex",
    gap: "2rem",
    alignItems: "flex-start",
    boxSizing: "border-box",
});

export const DocContent = styled("div")({
    flex: 1,
    minWidth: 0,
    "& h1": {
        fontSize: "2rem",
        fontWeight: 700,
        color: theme.colors.text.strongest,
        marginTop: 0,
        marginBottom: "1rem",
        lineHeight: 1.2,
    },
    "& h2": {
        fontSize: "1.35rem",
        fontWeight: 600,
        color: theme.colors.text.stronger,
        marginTop: "2.5rem",
        marginBottom: "0.75rem",
        paddingBottom: "0.4rem",
        borderBottom: `1px solid ${theme.colors.border.subtle}`,
        scrollMarginTop: "4.5rem",
    },
    "& h3": {
        fontSize: "1.05rem",
        fontWeight: 600,
        color: theme.colors.text.strong,
        marginTop: "1.75rem",
        marginBottom: "0.5rem",
        scrollMarginTop: "4.5rem",
    },
    "& h4": {
        fontSize: "0.9rem",
        fontWeight: 600,
        color: theme.colors.text.muted2,
        marginTop: "1.25rem",
        marginBottom: "0.4rem",
        textTransform: "uppercase",
        letterSpacing: "0.06em",
        scrollMarginTop: "4.5rem",
    },
    "& p": {
        marginTop: "0.75rem",
        marginBottom: "0.75rem",
        lineHeight: 1.7,
        color: theme.colors.text.primary,
    },
    "& ul, & ol": {
        paddingLeft: "1.5rem",
        margin: "0.75rem 0",
        lineHeight: 1.7,
    },
    "& li": {
        marginBottom: "0.3rem",
    },
    "& code": {
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
        fontSize: "0.85em",
        background: theme.colors.bg.panel,
        color: theme.colors.text.strong,
        padding: "0.15em 0.35em",
        borderRadius: theme.radii.sm,
        border: `1px solid ${theme.colors.border.subtle}`,
    },
    "& pre": {
        background: theme.colors.bg.panel,
        border: `1px solid ${theme.colors.border.subtle}`,
        borderRadius: theme.radii.md,
        padding: "1rem",
        overflowX: "auto",
        margin: "1rem 0",
    },
    "& pre code": {
        background: "none",
        border: "none",
        padding: 0,
        fontSize: "0.85rem",
        lineHeight: 1.6,
        color: theme.colors.text.primary,
    },
    "& blockquote": {
        borderLeft: `3px solid ${theme.colors.accent.cream}`,
        margin: "1rem 0",
        padding: "0.5rem 1rem",
        color: theme.colors.text.muted2,
        fontStyle: "italic",
    },
    "& table": {
        width: "100%",
        borderCollapse: "collapse",
        margin: "1rem 0",
        fontSize: "0.9rem",
    },
    "& th, & td": {
        padding: "0.5rem 0.75rem",
        textAlign: "left",
        borderBottom: `1px solid ${theme.colors.border.subtle}`,
    },
    "& th": {
        color: theme.colors.text.muted2,
        fontWeight: 600,
        fontSize: "0.8rem",
        textTransform: "uppercase",
        letterSpacing: "0.05em",
    },
    "& a": {
        color: theme.colors.accent.cream,
        textDecoration: "none",
        "&:hover": {
            textDecoration: "underline",
        },
    },
    "& hr": {
        border: "none",
        borderTop: `1px solid ${theme.colors.border.subtle}`,
        margin: "2rem 0",
    },
});

export const DocSidebar = styled("aside")({
    width: "200px",
    flexShrink: 0,
    position: "sticky",
    top: "4.5rem",
    maxHeight: "calc(100vh - 5rem)",
    overflowY: "auto",
    paddingRight: "0.5rem",
    "@media (max-width: 768px)": {
        display: "none",
    },
});

export const DocTocTitle = styled("div")({
    fontSize: "0.72rem",
    fontWeight: 600,
    textTransform: "uppercase",
    letterSpacing: "0.08em",
    color: theme.colors.text.muted,
    marginBottom: "0.5rem",
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
});

export const DocTocList = styled("ul")({
    listStyle: "none",
    padding: 0,
    margin: 0,
});

export const DocTocItem = styled("li")<{ level?: number }>({
    margin: "0.1rem 0",
    variants: [
        { props: { level: 3 }, style: { paddingLeft: "0.75rem" } },
        { props: { level: 4 }, style: { paddingLeft: "1.5rem" } },
    ],
});

export const DocTocLink = styled("a")({
    display: "block",
    fontSize: "0.78rem",
    color: theme.colors.text.muted,
    textDecoration: "none",
    padding: "0.2rem 0.4rem",
    borderRadius: theme.radii.sm,
    lineHeight: 1.4,
    "&:hover": {
        color: theme.colors.text.primary,
        background: theme.colors.hover.row,
    },
});

export const DocFooter = styled("footer")({
    borderTop: `1px solid ${theme.colors.border.subtle}`,
    padding: "2rem 1.5rem",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    gap: "0.75rem",
});

export const DocFooterNav = styled("nav")({
    display: "flex",
    flexWrap: "wrap",
    justifyContent: "center",
    gap: "0 1.5rem",
});

export const DocFooterNavLink = styled("a")({
    fontSize: "0.8rem",
    color: theme.colors.text.muted,
    textDecoration: "none",
    "&:hover": {
        color: theme.colors.text.primary,
    },
});

export const DocFooterCredit = styled("p")({
    margin: 0,
    fontSize: "0.75rem",
    color: theme.colors.text.muted,
    "& a": {
        color: theme.colors.text.muted2,
        textDecoration: "none",
        "&:hover": {
            color: theme.colors.text.primary,
        },
    },
});
