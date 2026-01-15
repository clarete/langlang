// @ts-nocheck
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const sourceDir = path.resolve(__dirname, '../../web/src/examples');
const outDir = process.env.LANGLANG_WEB_OUTDIR
  ? path.resolve(process.env.LANGLANG_WEB_OUTDIR)
  : path.resolve(__dirname, '../dist');
const destDir = path.resolve(outDir, 'assets/examples');

console.log(`Copying grammars from ${sourceDir} to ${destDir}...`);

if (!fs.existsSync(sourceDir)) {
  console.error(`Source directory not found: ${sourceDir}`);
  process.exit(1);
}

// Ensure dist directory exists
const distParent = path.dirname(destDir);
if (!fs.existsSync(distParent)) {
  fs.mkdirSync(distParent, { recursive: true });
}

try {
  const copyTree = (src: string, dst: string) => {
    const st = fs.lstatSync(src);
    if (st.isDirectory()) {
      fs.mkdirSync(dst, { recursive: true });
      for (const ent of fs.readdirSync(src)) {
        copyTree(path.join(src, ent), path.join(dst, ent));
      }
      return;
    }
    if (st.isSymbolicLink()) {
      const link = fs.readlinkSync(src);
      const target = path.resolve(path.dirname(src), link);
      const buf = fs.readFileSync(target);
      fs.mkdirSync(path.dirname(dst), { recursive: true });
      fs.writeFileSync(dst, buf);
      return;
    }
    fs.mkdirSync(path.dirname(dst), { recursive: true });
    fs.copyFileSync(src, dst);
  };

  copyTree(sourceDir, destDir);
  console.log('Grammars copied successfully.');
} catch (error) {
  console.error('Failed to copy grammars:', error);
  process.exit(1);
}

