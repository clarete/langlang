#!/usr/bin/env node
import { Command } from 'commander';
import packageJson from '../package.json' with { type: 'json' };
import path from 'node:path';
import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const program = new Command();

program
  .name('langlang')
  .description('CLI for @langlang')
  .version(packageJson.version);

program.command('install')
  .description('Install the LangLang WASM binary')
  .option('-d, --destination <path>', 'Destination directory', process.cwd())
  .action(async ({ destination }: { destination: string }) => {
    console.log('Installing the LangLang WASM binary');

    const wasmBinUrl = process.env.LANG_LANG_WASM_BIN_URL;
    const wasmExecUrl = process.env.GO_WASM_EXEC_JS_URL;

    if (!wasmBinUrl) {
      console.error('LANG_LANG_WASM_BIN_URL is not set');
      process.exit(1);
    }

    if (!wasmExecUrl) {
      console.error('LANG_LANG_WASM_EXEC_URL is not set');
      process.exit(1);
    }

    console.log(`Fetching exec from: ${wasmExecUrl}`);
    console.log(`Fetching binary from: ${wasmBinUrl}`);

    const download = async (url: string): Promise<Buffer> => {
      if (url.startsWith('file://')) {
        const filePath = fileURLToPath(url);
        return fs.readFile(filePath);
      }

      if (url.startsWith('http://') || url.startsWith('https://')) {
        const res = await fetch(url);
        if (!res.ok) throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
        return Buffer.from(await res.arrayBuffer());

      }


      throw new Error(`Unsupported URL scheme: ${url}`);
    }

    try {
      const wasmExecPath = path.join(destination, 'wasm_exec.js');
      const wasmExecBuf = await download(wasmExecUrl);
      await fs.writeFile(wasmExecPath, wasmExecBuf);
      console.log('✅ WASM exec installed');

      const wasmBinPath = path.join(destination, 'langlang.wasm');
      const wasmBinBuf = await download(wasmBinUrl);
      await fs.writeFile(wasmBinPath, wasmBinBuf);
      console.log('✅ WASM binary installed');

      console.log('LangLang binaries and exec are now installed, add the following to the end of your HTML body file:')
      console.log(`<script src="${path.relative(process.cwd(), wasmExecPath)}"></script>`);
    } catch (err) {
      console.error('Error installing files:', err);
      process.exit(1);
    }

  });


program.parse();
