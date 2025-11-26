import dts from 'bun-plugin-dts';
import dotenv from 'dotenv';

dotenv.config();

console.log("Building with environment variables:");
console.log("LANG_LANG_WASM_BIN_URL:", process.env.LANG_LANG_WASM_BIN_URL);
console.log("GO_WASM_EXEC_JS_URL:", process.env.GO_WASM_EXEC_JS_URL);

await Bun.build({
  entrypoints: ['./src/index.ts'],
  outdir: './dist',
  target: 'node',
  plugins: [
    dts()
  ],
  define: {
    'process.env.LANG_LANG_WASM_BIN_URL': JSON.stringify(process.env.LANG_LANG_WASM_BIN_URL ?? ""),
    'process.env.GO_WASM_EXEC_JS_URL': JSON.stringify(process.env.GO_WASM_EXEC_JS_URL ?? ""),
  },
});
