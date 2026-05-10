export const theme = {
    fonts: {
        mono: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    },
    colors: {
        bg: {
            app: "var(--bg-app)",
            panel: "var(--bg-panel)",
            editor: "var(--bg-editor)",
            sidebar: "var(--bg-sidebar)",
            bar: "var(--bg-bar)",
            control: "var(--bg-control)",
            controlHover: "var(--bg-control-hover)",
            controlHoverStrong: "var(--bg-control-hover-strong)",
            overlayDark: "var(--bg-overlay-dark)",
        },
        text: {
            primary: "var(--text-primary)",
            strong: "var(--text-strong)",
            stronger: "var(--text-stronger)",
            strongest: "var(--text-strongest)",
            muted: "var(--text-muted)",
            muted2: "var(--text-muted2)",
            placeholder: "var(--text-placeholder)",
        },
        border: {
            subtle: "var(--border-subtle)",
            subtle2: "var(--border-subtle2)",
            normal: "var(--border-normal)",
            normal2: "var(--border-normal2)",
            tab: "var(--border-tab)",
            hover: "var(--border-hover)",
            hover2: "var(--border-hover2)",
            active: "var(--border-active)",
            active2: "var(--border-active2)",
        },
        hover: {
            row: "var(--hover-row)",
            control: "var(--hover-control)",
        },
        accent: {
            cream: "var(--accent-cream)",
            creamHover: "var(--accent-cream-hover)",
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
