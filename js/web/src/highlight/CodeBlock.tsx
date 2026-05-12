import { useEffect, useMemo, useState } from "react";
import { highlight, isSupported } from "./highlighter";
import { makeRenderer } from "./renderHighlight";

interface CodeProps {
    className?: string;
    children?: React.ReactNode;
}

export function CodeBlock({ className, children }: CodeProps) {
    const lang = className?.replace("language-", "") ?? "";
    const code = String(children ?? "").replace(/\n$/, "");

    if (isSupported(lang)) {
        return <HighlightedCode lang={lang} code={code} />;
    }

    return <code className={className}>{children}</code>;
}

function HighlightedCode({ lang, code }: { lang: string; code: string }) {
    const [highlighted, setHighlighted] = useState<React.ReactNode>(null);
    const render = useMemo(() => makeRenderer(lang), [lang]);

    useEffect(() => {
        let cancelled = false;
        highlight(lang, code)
            .then((v) => {
                if (!cancelled) setHighlighted(v.type === "error" ? null : (render(v, "root") ?? null));
            })
            .catch(() => {});
        return () => { cancelled = true; };
    }, [lang, code, render]);

    return (
        <code className={`language-${lang} ll-highlighted`}>
            {highlighted ?? code}
        </code>
    );
}
