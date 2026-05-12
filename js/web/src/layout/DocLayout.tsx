import { useEffect, useRef, useState } from "react";
import { useLocation } from "react-router-dom";
import { MDXProvider } from "@mdx-js/react";
import NavBar from "../nav/NavBar";
import { CodeBlock } from "../highlight/CodeBlock";
import SiteFooter from "./SiteFooter";
import {
    DocRoot,
    DocMain,
    DocContent,
    DocSidebar,
    DocTocTitle,
    DocTocList,
    DocTocItem,
    DocTocLink,
} from "./DocLayout.styles";

function CopyableCodeBlock(props: React.HTMLAttributes<HTMLPreElement>) {
    const [copied, setCopied] = useState(false);
    const ref = useRef<HTMLPreElement>(null);

    function copy() {
        const text = ref.current?.querySelector("code")?.innerText ?? "";
        navigator.clipboard.writeText(text).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        });
    }

    return (
        <div style={{ position: "relative" }} className="code-block-wrapper">
            <pre ref={ref} {...props} />
            <button onClick={copy} className="copy-button" aria-label="Copy code">
                {copied ? "✓" : "Copy"}
            </button>
        </div>
    );
}

function makeHeading(Tag: "h1" | "h2" | "h3" | "h4") {
    return function Heading({ id, children, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
        return (
            <Tag id={id} {...props}>
                {children}
                {id && (
                    <a href={`#${id}`} className="heading-anchor" aria-hidden="true" tabIndex={-1}>
                        #
                    </a>
                )}
            </Tag>
        );
    };
}

const mdxComponents = {
    code: CodeBlock,
    pre: CopyableCodeBlock,
    h2: makeHeading("h2"),
    h3: makeHeading("h3"),
    h4: makeHeading("h4"),
};

interface TocEntry {
    id: string;
    text: string;
    level: number;
}

function TableOfContents({ entries, activeId }: { entries: TocEntry[]; activeId: string }) {
    if (entries.length === 0) return null;
    return (
        <DocSidebar>
            <DocTocTitle>On this page</DocTocTitle>
            <DocTocList>
                {entries.map((e) => (
                    <DocTocItem key={e.id} level={e.level}>
                        <DocTocLink href={`#${e.id}`} data-active={e.id === activeId ? "true" : undefined}>{e.text}</DocTocLink>
                    </DocTocItem>
                ))}
            </DocTocList>
        </DocSidebar>
    );
}

export default function DocLayout({ children }: { children: React.ReactNode }) {
    const location = useLocation();
    const [toc, setToc] = useState<TocEntry[]>([]);
    const [activeId, setActiveId] = useState<string>("");

    useEffect(() => {
        const headings = Array.from(
            document.querySelectorAll<HTMLHeadingElement>(
                ".doc-content h2, .doc-content h3, .doc-content h4",
            ),
        );
        setToc(
            headings
                .filter((h) => h.id)
                .map((h) => {
                    const clone = h.cloneNode(true) as HTMLHeadingElement;
                    clone.querySelector(".heading-anchor")?.remove();
                    return {
                        id: h.id,
                        text: clone.textContent ?? "",
                        level: parseInt(h.tagName[1], 10),
                    };
                }),
        );
        setActiveId("");
    }, [location.pathname]);

    useEffect(() => {
        if (toc.length === 0) return;

        function update() {
            const headings = Array.from(
                document.querySelectorAll<HTMLElement>(
                    ".doc-content h2, .doc-content h3, .doc-content h4",
                ),
            ).filter((h) => h.id);
            if (headings.length === 0) return;
            // Nav bar is ~3.5rem tall; activate a heading once it reaches that line
            const threshold = 64;
            let active = headings[0].id;
            for (const h of headings) {
                if (h.getBoundingClientRect().top <= threshold) active = h.id;
            }
            setActiveId(active);
        }

        update();
        window.addEventListener("scroll", update, { passive: true });
        return () => window.removeEventListener("scroll", update);
    }, [toc]);

    return (
        <MDXProvider components={mdxComponents}>
            <DocRoot>
                <NavBar />
                <DocMain>
                    <TableOfContents entries={toc} activeId={activeId} />
                    <DocContent className="doc-content">{children}</DocContent>
                </DocMain>
                <SiteFooter />
            </DocRoot>
        </MDXProvider>
    );
}
