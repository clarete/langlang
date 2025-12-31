import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { pigment } from '@pigment-css/vite-plugin'

export default defineConfig({
  plugins: [pigment(), react()],
  optimizeDeps: {
    include: ['react-is', '@pigment-css/react', '@monaco-editor/react']
  },
  build: {
    target: 'esnext'
  }
})
