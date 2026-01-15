import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { pigment } from '@pigment-css/vite-plugin'

export default defineConfig({
  base: process.env.LANGLANG_WEB_BASE ?? '/',
  plugins: [react(), pigment({})],
  optimizeDeps: {
    include: ['react-is', '@pigment-css/react', '@monaco-editor/react']
  },
  build: {
    target: 'esnext',
    outDir: process.env.LANGLANG_WEB_OUTDIR ?? 'dist'
  }
})
