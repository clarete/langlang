// @ts-nocheck
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const sourceDir = path.resolve(__dirname, '../../../grammars');
const destDir = path.resolve(__dirname, '../dist/assets/grammars');

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

// Copy recursively
try {
  fs.cpSync(sourceDir, destDir, { recursive: true, force: true });
  console.log('Grammars copied successfully.');
} catch (error) {
  console.error('Failed to copy grammars:', error);
  process.exit(1);
}

