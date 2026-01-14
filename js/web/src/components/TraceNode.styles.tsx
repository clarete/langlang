import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const NodeContainer = styled("div")<{ leaf?: boolean }>({
    display: "flex",
    flexDirection: "column",
    gap: ".2rem",
    variants: [
        {
            props: ({ leaf }) => leaf === undefined || leaf === false,
            style: {
                width: "fit-content",
            },
        },
    ],
});

export const SequenceContainer = styled("div")({
    display: "grid",
    gridTemplateColumns: "repeat(var(--count, 1), 1fr)",
    gap: ".2rem",
});

export const NodeName = styled("div")<{
    highlighted?: boolean;
    parentHighlighted?: boolean;
    level?: number;
}>({
    backgroundColor: "hsl(36, 78%, calc(var(--highlight-level, 100) * 0.93%))",
    color: theme.colors.bg.panel,
    padding: "0.3rem 0.7rem",
    fontWeight: 500,
    fontSize: "calc(0.5em * var(--line-height))",
    margin: 0,
    whiteSpace: "nowrap",
    position: "relative",
    borderRadius: "2px",
    cursor: "pointer",
    minHeight: "calc(var(--line-height) * 1em)",
    variants: [
        {
            props: { highlighted: true },
            style: {
                backgroundColor: "hsl(38, 77%, 80%)",
                color: "hsl(0, 0%, 0%)",
            },
        },
        {
            props: { highlighted: true, level: 0 },
            style: {
                backgroundColor: "hsl(103deg 44% 40%)",
                color: "hsl(38, 76%, 95%)",
            },
        },
        {
            props: { highlighted: true, level: 1 },
            style: { backgroundColor: "hsl(38, 76%, 40%)" },
        },
        {
            props: { highlighted: true, level: 2 },
            style: { backgroundColor: "hsl(38, 76%, 50%)" },
        },
        {
            props: { highlighted: true, level: 3 },
            style: { backgroundColor: "hsl(38, 76%, 60%)" },
        },
        {
            props: { highlighted: true, level: 4 },
            style: { backgroundColor: "hsl(38, 76%, 70%)" },
        },
        {
            props: { parentHighlighted: true },
            style: {
                backgroundColor: "hsl(218, 33%, 50%)",
                color: "hsl(0, 0%, 100%)",
            },
        },
        {
            props: { parentHighlighted: true, level: 0 },
            style: { backgroundColor: "hsl(218, 33%, 5%)" },
        },
        {
            props: { parentHighlighted: true, level: 1 },
            style: { backgroundColor: "hsl(218, 33%, 10%)" },
        },
        {
            props: { parentHighlighted: true, level: 2 },
            style: { backgroundColor: "hsl(218, 33%, 20%)" },
        },
        {
            props: { parentHighlighted: true, level: 3 },
            style: { backgroundColor: "hsl(218, 33%, 30%)" },
        },
        {
            props: { parentHighlighted: true, level: 4 },
            style: { backgroundColor: "hsl(218, 33%, 40%)" },
        },
    ],
});
