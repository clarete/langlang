import { styled } from "@pigment-css/react";

export const TreeViewContainer = styled("div")({
    flex: 1,
    display: "flex",
    flexDirection: "column",
    gap: "0.5rem",
    // width: "fit-content",
});

export const SourceLine = styled("div")({
    display: "flex",
    // gridTemplateColumns: "repeat(auto-fit, minmax(0, 1fr))",
    gap: "0.5rem",
});