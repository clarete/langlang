import { styled } from "@pigment-css/react";

export const TraceViewContainer = styled("div")({
    flex: 1,
    display: "flex",
    flexDirection: "column",
    gap: "0.2rem",
    overflowX: "auto",
    padding: "0.75rem",
});

export const SourceLine = styled("div")({
    display: "flex",
    gap: "0.2rem",
});
