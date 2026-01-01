import { styled } from "@pigment-css/react";

export const TraceViewContainer = styled("div")({
    flex: 1,
    display: "flex",
    flexDirection: "column",
    gap: "0.2rem",
    // width: "fit-content",
});

export const SourceLine = styled("div")({
    display: "flex",
    // gridTemplateColumns: "repeat(auto-fit, minmax(0, 1fr))",
    gap: "0.2rem",
});
