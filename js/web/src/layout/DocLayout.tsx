import { useLayoutEffect, useState } from "react";
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

interface TocEntry {
    id: string;
    text: string;
    level: number;
}

function TableOfContents({ entries }: { entries: TocEntry[] }) {
    if (entries.length === 0) return null;
    return (
        <DocSidebar>
            <DocTocTitle>On this page</DocTocTitle>
            <DocTocList>
                {entries.map((e) => (
                    <DocTocItem key={e.id} level={e.level}>
                        <DocTocLink href={`#${e.id}`}>{e.text}</DocTocLink>
                    </DocTocItem>
                ))}
            </DocTocList>
        </DocSidebar>
    );
}

export default function DocLayout({ children }: { children: React.ReactNode }) {
    const location = useLocation();
    const [toc, setToc] = useState<TocEntry[]>([]);

    useLayoutEffect(() => {
        const headings = Array.from(
            document.querySelectorAll<HTMLHeadingElement>(
                ".doc-content h2, .doc-content h3, .doc-content h4",
            ),
        );
        setToc(
            headings
                .filter((h) => h.id)
                .map((h) => ({
                    id: h.id,
                    text: h.textContent ?? "",
                    level: parseInt(h.tagName[1], 10),
                })),
        );
    }, [location.pathname]);

    return (
        <MDXProvider components={{ code: CodeBlock }}>
            <DocRoot>
                <NavBar />
                <DocMain>
                    <TableOfContents entries={toc} />
                    <DocContent className="doc-content">{children}</DocContent>
                </DocMain>
                <SiteFooter />
            </DocRoot>
        </MDXProvider>
    );
}
