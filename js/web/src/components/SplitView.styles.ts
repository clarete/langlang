import { styled } from "@pigment-css/react";

export const SplitViewRoot = styled("div")<{ horizontal: boolean }>({
    display: "flex",
    flexGrow: 0,
    flexShrink: 0,
    flexBasis: "100%",
    position: "relative",
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

export const SplitViewContainer = styled("div")<{ span: number, horizontal: boolean }>({
    flex: ({ span }) => span,
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",

    variants: [
        {
            props: { horizontal: true },
            style: {
                overflowX: "hidden",
                overflowY: "auto",
            },
        },
        {
            props: { horizontal: false },
            style: {
                overflowX: "auto",
                overflowY: "hidden",
            },
        },
    ],

});

export const SplitViewHandleKnob = styled("div")<{ horizontal: boolean }>({
    width: "0.25rem",
    height: "1.5rem",
    backgroundColor: '#fff',
    borderRadius: "0.25rem",
    boxShadow: "0 0 0 1px #000",
    cursor: "grab",
    zIndex: 1000,

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

export const SplitViewHandle = styled("div")<{ span: number, horizontal: boolean }>({
    flex: 0,
    position: "absolute",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    transition: "background-color 0.2s ease-in-out",
    backgroundColor: 'hsl(0deg 0% 14.12%)',
    borderRadius: "0.25rem",
    cursor: "grab",
    

    '&:hover': {
        backgroundColor: 'hsl(0deg 0% 20%)',
    },

    variants: [
        {
            props: { horizontal: false },
            style: {
                width: "0.5rem",
                top: 0,
                bottom: 0,
                left: ({ span }) => `calc(${span * 100}% - 0.25rem)`,
                right: ({ span }) => `calc(${(1 - span) * 100}% - 0.25rem)`,

                [`&:hover ${SplitViewHandleKnob}`]: {
                    width: "0.5rem",
                },
            },
        },
        {
            props: { horizontal: true },
            style: {
                width: '100%',
                height: "0.5rem",
                left: 0,
                right: 0,
                top: ({ span }) => `calc(${span * 100}% - 0.25rem)`,
                bottom: ({ span }) => `calc(${(1 - span) * 100}% - 0.25rem)`,
                [`&:hover ${SplitViewHandleKnob}`]: {
                    height: "0.5rem",
                },
            },
        }
    ],
});

