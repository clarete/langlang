// @ts-nocheck
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const outDir = process.env.LANGLANG_WEB_OUTDIR
    ? path.resolve(process.env.LANGLANG_WEB_OUTDIR)
    : path.resolve(__dirname, "../dist");

const serverBuildDir = path.resolve(__dirname, "../dist-server");
const { render } = await import(path.join(serverBuildDir, "entry-server.js"));

const template = readFileSync(path.join(outDir, "index.html"), "utf-8");

const routes = [
    "/",
    "/getting-started",
    "/language-reference",
    "/examples",
    "/playground",
];

for (const route of routes) {
    const appHtml = render(route);
    const html = template.replace(
        '<div id="app"></div>',
        `<div id="app">${appHtml}</div>`,
    );
    const outFile =
        route === "/"
            ? path.join(outDir, "index.html")
            : path.join(outDir, route.slice(1), "index.html");
    mkdirSync(path.dirname(outFile), { recursive: true });
    writeFileSync(outFile, html);
    console.log(`prerendered: ${route} → ${path.relative(outDir, outFile)}`);
}
