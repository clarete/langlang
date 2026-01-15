import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import { spawnSync } from "node:child_process"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(__dirname, "../..")

const base = `live/`
const outDir = path.resolve(path.join(repoRoot, "docs", "live"))

const env = {
    ...process.env,
    LANGLANG_WEB_BASE: base,
    LANGLANG_WEB_OUTDIR: outDir,
}

console.log(`[pages] base=${env.LANGLANG_WEB_BASE}`)
console.log(`[pages] outDir=${env.LANGLANG_WEB_OUTDIR}`)

function safeRecursiveDelete(p: string) {
    if (!fs.existsSync(p)) {
        return
    }
    const st = fs.lstatSync(p)
    if (st.isDirectory()) {
        for (const ent of fs.readdirSync(p)) {
            safeRecursiveDelete(path.join(p, ent))
        }
        fs.rmdirSync(p)
        return
    }
    // Do not follow symlinks; just remove the link itself.
    fs.unlinkSync(p)
}

try {
    safeRecursiveDelete(outDir)
    fs.mkdirSync(outDir, { recursive: true })
} catch (err) {
    console.warn(
        `[pages] warning: failed to clean ${outDir}; continuing without cleaning`,
        err,
    )
}

const res = spawnSync(
    "bun",
    ["run", "--filter", "@langlang/web-test", "build"],
    {
        cwd: path.join(repoRoot, "js"),
        stdio: "inherit",
        env,
    },
)

process.exit(res.status ?? 1)
