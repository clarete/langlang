import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const NavRoot = styled("nav")({
    position: "sticky",
    top: 0,
    zIndex: 100,
    background: theme.colors.bg.bar,
    borderBottom: `1px solid ${theme.colors.border.normal}`,
    height: "3.5rem",
    display: "flex",
    alignItems: "center",
});

export const NavContainer = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0",
    padding: "0 1.5rem",
    width: "100%",
    maxWidth: "72rem",
    margin: "0 auto",
});

export const NavLogo = styled("a")({
    fontFamily: theme.fonts.mono,
    fontWeight: 700,
    fontSize: "1rem",
    color: theme.colors.accent.cream,
    textDecoration: "none",
    letterSpacing: "0.02em",
    marginRight: "auto",
    "&:hover": {
        color: theme.colors.accent.creamHover,
    },
});

export const NavLinks = styled("div")({
    display: "flex",
    alignItems: "center",
    gap: "0.25rem",
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
