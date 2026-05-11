import ReactDOMServer from "react-dom/server";
import { StaticRouter, Routes, Route } from "react-router";
import DocLayout from "./layout/DocLayout";
import PlaygroundLayout from "./layout/PlaygroundLayout";
import PlaygroundPage from "./pages/PlaygroundPage";
import HomePage from "./pages/HomePage";
import GettingStarted from "./docs/getting-started.mdx";
import LanguageReference from "./docs/language-reference.mdx";
import Examples from "./docs/examples.mdx";

export function render(url: string): string {
    const basename = import.meta.env.BASE_URL.replace(/\/$/, "") || "/";
    return ReactDOMServer.renderToString(
        <StaticRouter location={url} basename={basename}>
            <Routes>
                <Route path="/" element={<DocLayout><HomePage /></DocLayout>} />
                <Route path="/getting-started" element={<DocLayout><GettingStarted /></DocLayout>} />
                <Route path="/language-reference" element={<DocLayout><LanguageReference /></DocLayout>} />
                <Route path="/examples" element={<DocLayout><Examples /></DocLayout>} />
                <Route path="/playground" element={<PlaygroundLayout><PlaygroundPage /></PlaygroundLayout>} />
            </Routes>
        </StaticRouter>,
    );
}
