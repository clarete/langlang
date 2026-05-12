import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const NavRoot = styled("nav")({
    position: "sticky",
    top: 0,
    zIndex: 100,
    background: theme.colors.bg.bar,
    borderBottom: `1px solid ${theme.colors.border.normal}`,
});

export const NavContainer = styled("div")({
    display: "flex",
    alignItems: "center",
    padding: "0 1.5rem",
    width: "100%",
    maxWidth: "72rem",
    margin: "0 auto",
    height: "3.5rem",
    minWidth: 0,
    boxSizing: "border-box",
});

export const NavLogo = styled("a")({
    fontFamily: theme.fonts.mono,
    fontWeight: 700,
    fontSize: "1rem",
    color: theme.colors.accent.cream,
    textDecoration: "none",
    letterSpacing: "0.02em",
    marginRight: "auto",
    whiteSpace: "nowrap",
    flexShrink: 0,
    "&:hover": {
        color: theme.colors.accent.creamHover,
    },
});

export const NavLinks = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.25rem",
    flexShrink: 0,
    "@media (max-width: 768px)": {
        display: "none",
    },
});

export const HamburgerButton = styled("button")({
    display: "none",
    alignItems: "center",
    justifyContent: "center",
    width: "28px",
    height: "28px",
    flexShrink: 0,
    marginLeft: "0.5rem",
    background: "transparent",
    border: "none",
    borderRadius: theme.radii.md,
    color: theme.colors.text.muted2,
    cursor: "pointer",
    fontSize: "1rem",
    lineHeight: 1,
    padding: 0,
    transition: "color 0.12s, background 0.12s",
    "&:hover": {
        color: theme.colors.text.strong,
        background: theme.colors.hover.control,
    },
    "@media (max-width: 768px)": {
        display: "flex",
    },
});

export const MobileMenu = styled("div")({
    display: "flex",
    flexDirection: "column",
    background: theme.colors.bg.bar,
    borderTop: `1px solid ${theme.colors.border.normal}`,
    padding: "0.5rem 1rem",
});

export const MobileNavLink = styled("a")({
    fontFamily: theme.fonts.mono,
    fontSize: "0.9rem",
    color: theme.colors.text.muted2,
    textDecoration: "none",
    padding: "0.65rem 0.5rem",
    borderBottom: `1px solid ${theme.colors.border.subtle}`,
    transition: "color 0.12s",
    "&:last-child": {
        borderBottom: "none",
    },
    "&:hover": {
        color: theme.colors.accent.cream,
    },
});

export const NavLink = styled("a")({
    fontFamily: theme.fonts.mono,
    fontSize: "0.82rem",
    color: theme.colors.text.muted2,
    textDecoration: "none",
    padding: "0.35rem 0.7rem",
    borderRadius: theme.radii.md,
    transition: "color 0.12s, background 0.12s",
    "&:hover": {
        color: theme.colors.accent.cream,
        background: theme.colors.hover.control,
    },
});

export const ThemeToggleButton = styled("button")({
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    width: "28px",
    height: "28px",
    flexShrink: 0,
    marginLeft: "0.25rem",
    background: "transparent",
    border: "none",
    borderRadius: theme.radii.md,
    color: theme.colors.text.muted2,
    cursor: "pointer",
    fontSize: "0.9rem",
    lineHeight: 1,
    padding: 0,
    transition: "color 0.12s, background 0.12s",
    "&:hover": {
        color: theme.colors.text.strong,
        background: theme.colors.hover.control,
    },
});
