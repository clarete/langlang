import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const HomeMain = styled("div")({
    width: "100%",
    maxWidth: "72rem",
    margin: "0 auto",
    padding: "2rem 1.5rem 3rem",
    boxSizing: "border-box",
    display: "flex",
    flexDirection: "column",
    gap: "2rem",
});

export const HomeIntro = styled("div")({
    maxWidth: "44rem",
});

export const HomeDescription = styled("p")({
    margin: "1rem 0",
    fontSize: "1rem",
    lineHeight: 1.7,
    color: theme.colors.text.muted2,
});

export const HomeFeatures = styled("ul")({
    margin: 0,
    padding: "0 0 0 1.25rem",
    listStyle: "disc",
    "& li": {
        fontSize: "0.9rem",
        lineHeight: 1.7,
        color: theme.colors.text.muted,
        marginBottom: "0.2rem",
    },
});
