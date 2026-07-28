import { resolve } from 'node:path'
import { defineConfig } from 'vite'

export default defineConfig({
  root: resolve(__dirname, 'mobile-viewer'),
  base: './',
  publicDir: false,
  build: {
    outDir: resolve(__dirname, 'dist-mobile-viewer'),
    emptyOutDir: true,
    sourcemap: false,
    manifest: true,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/viewer-[hash].js',
        chunkFileNames: 'assets/chunk-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]'
      }
    }
  },
  resolve: {
    conditions: ['browser', 'import', 'default']
  }
})
