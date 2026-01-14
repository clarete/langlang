import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const SettingsPanel = styled("div")({
    padding: "0.6rem 0.75rem",
    background: theme.colors.bg.control,
    border: `1px solid ${theme.colors.border.normal}`,
    borderTop: "none",
    marginTop: "-1px",
});

export const SettingsRow = styled("label")({
    display: "flex",
    alignItems: "center",
    gap: "0.55rem",
    padding: "0.35rem 0",
    fontFamily: theme.fonts.mono,
    fontSize: "0.78rem",
    color: theme.colors.text.primary,
    userSelect: "none",
});
