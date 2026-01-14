export function fg(alpha: number) {
    return `rgba(251, 240, 223, ${alpha})`;
}

export const theme = {
    fonts: {
        mono: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    },
    colors: {
        bg: {
            app: "#242424",
            panel: "#1a1a1a",
            // Monaco `vs-dark` editor surface background.
            editor: "#1e1e1e",
            sidebar: "#161616",
            bar: "#121212",
            control: "#242424",
            controlHover: "#303030",
            controlHoverStrong: "#6b6b6b",
            overlayDark: "rgba(0, 0, 0, 0.12)",
        },
        text: {
            primary: fg(0.85),
            strong: fg(0.9),
            stronger: fg(0.92),
            strongest: fg(0.98),
            muted: fg(0.55),
            muted2: fg(0.75),
            placeholder: fg(0.4),
        },
        border: {
            subtle: fg(0.12),
            subtle2: fg(0.14),
            normal: fg(0.18),
            normal2: fg(0.2),
            tab: fg(0.15),
            hover: fg(0.25),
            hover2: fg(0.22),
            active: fg(0.28),
            active2: fg(0.35),
        },
        hover: {
            row: "rgba(255, 255, 255, 0.06)",
            control: "rgba(255, 255, 255, 0.08)",
        },
        accent: {
            cream: "#fbf0df",
            creamHover: "#f3d5a3",
        },
        solid: {
            white: "#fff",
            black: "#000",
        },
    },
    radii: {
        xs: "2px",
        sm: "4px",
        md: "6px",
        lg: "8px",
        xl: "12px",
    },
} as const;
