import { styled } from "@pigment-css/react";
import { theme } from "../theme";

export const SplitViewRoot = styled("div")<{ horizontal: boolean }>({
    display: "flex",
    position: "relative",
    flex: "1 1 auto",
    minWidth: 0,
    minHeight: 0,
    overflow: "hidden",
    variants: [
        {
            props: { horizontal: true },
            style: {
                flexDirection: "column",
            },
        },
        {
            props: { horizontal: false },
            style: {
                flexDirection: "row",
            },
        },
    ],
});

export const SplitViewContainer = styled("div")<{
    span: number;
    horizontal: boolean;
}>({
    flex: ({ span }) => span,
    display: "flex",
    flexDirection: "column",
    alignItems: "stretch",
    minWidth: 0,
    minHeight: 0,

    variants: [
        {
            props: { horizontal: true },
            style: {
                overflowX: "hidden",
                overflowY: "auto",
                minHeight: 0,
            },
        },
        {
            props: { horizontal: false },
            style: {
                maxWidth: "100%",
            },
        },
    ],
});

export const SplitViewHandleKnob = styled("div")<{ horizontal: boolean }>({
    width: "0.25rem",
    height: "1.5rem",
    backgroundColor: theme.colors.solid.white,
    borderRadius: "0.25rem",
    boxShadow: `0 0 0 1px ${theme.colors.solid.black}`,
    cursor: "grab",
    zIndex: 1000,
    opacity: 0,
    transition: "opacity 0.12s ease-in-out",

    variants: [
        {
            props: { horizontal: false },
            style: {
                width: "0.25rem",
                height: "1.5rem",
                transition: "width 0.2s ease-in-out",
            },
        },
        {
            props: { horizontal: true },
            style: {
                width: "1.5rem",
                height: "0.25rem",
                transition: "height 0.2s ease-in-out",
            },
        },
    ],
});

export const SplitViewHandle = styled("div")<{
    span: number;
    horizontal: boolean;
}>({
    flex: 0,
    position: "absolute",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    // Keep a larger hit target for dragging, but render only a 1px
    // divider line by default.
    backgroundColor: "transparent",
    cursor: "grab",

    "&::before": {
        content: '""',
        position: "absolute",
        backgroundColor: theme.colors.border.subtle2,
        borderRadius: "1px",
        transition:
            "background-color 0.12s ease-in-out, width 0.12s ease-in-out, height 0.12s ease-in-out",
    },

    variants: [
        {
            props: { horizontal: false },
            style: {
                // Hit target
                width: "0.5rem",
                top: 0,
                bottom: 0,
                left: ({ span }) => `calc(${span * 100}% - 0.25rem)`,
                right: ({ span }) => `calc(${(1 - span) * 100}% - 0.25rem)`,

                "&::before": {
                    width: "1px",
                    top: 0,
                    bottom: 0,
                    left: "calc(50% - 0.5px)",
                },
                "&:hover::before": {
                    width: "3px",
                    left: "calc(50% - 1.5px)",
                    backgroundColor: theme.colors.border.hover,
                },
                [`&:hover ${SplitViewHandleKnob}`]: {
                    width: "0.5rem",
                    opacity: 1,
                },
            },
        },
        {
            props: { horizontal: true },
            style: {
                // Hit target
                width: "100%",
                height: "0.5rem",
                left: 0,
                right: 0,
                top: ({ span }) => `calc(${span * 100}% - 0.25rem)`,
                bottom: ({ span }) => `calc(${(1 - span) * 100}% - 0.25rem)`,
                "&::before": {
                    height: "1px",
                    left: 0,
                    right: 0,
                    top: "calc(50% - 0.5px)",
                },
                "&:hover::before": {
                    height: "3px",
                    top: "calc(50% - 1.5px)",
                    backgroundColor: theme.colors.border.hover,
                },
                [`&:hover ${SplitViewHandleKnob}`]: {
                    height: "0.5rem",
                    opacity: 1,
                },
            },
        },
    ],
});
