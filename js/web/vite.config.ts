import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { pigment } from "@pigment-css/vite-plugin";
import mdx from "@mdx-js/rollup";
import remarkGfm from "remark-gfm";
import rehypeSlug from "rehype-slug";
import rehypeAutolinkHeadings from "rehype-autolink-headings";

export default defineConfig({
    base: process.env.LANGLANG_WEB_BASE ?? "/",
    plugins: [
        {
            enforce: "pre",
            ...mdx({
                providerImportSource: "@mdx-js/react",
                remarkPlugins: [remarkGfm],
                rehypePlugins: [rehypeSlug, rehypeAutolinkHeadings],
            }),
        },
        react(),
        pigment({}),
    ],
    optimizeDeps: {
        include: ["react-is", "@pigment-css/react", "@monaco-editor/react"],
        // Don't try to optimize the WASM files
        exclude: ["@langlang/wasm"],
    },
    worker: {
        format: "es",
    },
    build: {
        target: "esnext",
        outDir: process.env.LANGLANG_WEB_OUTDIR ?? "dist",
    },
    // Ensure WASM files are served correctly
    assetsInclude: ["**/*.wasm"],
});
