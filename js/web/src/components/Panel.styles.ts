import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const PanelContainer = styled("div")({
    background: theme.colors.bg.panel,
    height: "100%",
    width: "100%",
    display: "flex",
    flexDirection: "column",
});

export const PanelHeader = styled("div")({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    color: theme.colors.text.primary,
    fontFamily: theme.fonts.mono,
    fontSize: "0.8rem",
    letterSpacing: "0.02em",
    userSelect: "none",
});

export const PanelBody = styled("div")({
    flex: 1,
    minHeight: 0,
});
