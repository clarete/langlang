import { useState, useEffect } from "react";
import { ThemeToggleButton } from "./NavBar.styles";

type ThemeMode = "light" | "dark";

function currentTheme(): ThemeMode {
    return (document.documentElement.getAttribute("data-theme") as ThemeMode) ?? "light";
}

export function applyTheme(mode: ThemeMode) {
    document.documentElement.setAttribute("data-theme", mode);
    localStorage.setItem("ll-theme", mode);
    window.dispatchEvent(new CustomEvent("theme-change", { detail: mode }));
}

export default function ThemeToggle() {
    const [mode, setMode] = useState<ThemeMode>("light");

    useEffect(() => { setMode(currentTheme()); }, []);

    useEffect(() => {
        const handler = (e: Event) => setMode((e as CustomEvent<ThemeMode>).detail);
        window.addEventListener("theme-change", handler);
        return () => window.removeEventListener("theme-change", handler);
    }, []);

    function toggle() {
        const next = mode === "dark" ? "light" : "dark";
        applyTheme(next);
        setMode(next);
    }

    return (
        <ThemeToggleButton onClick={toggle} aria-label={`Switch to ${mode === "dark" ? "light" : "dark"} mode`} title={`Switch to ${mode === "dark" ? "light" : "dark"} mode`}>
            {mode === "dark" ? "☀" : "☽"}
        </ThemeToggleButton>
    );
}
