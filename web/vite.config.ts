/// <reference types="vite/client" />

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve, join } from 'path'

const rootDir = resolve(__dirname)
const [distDir, srcDir] = [join(rootDir, 'dist'), join(rootDir, 'src')]
const isWatchMode = ['serve', 'dev', 'watch'].some((arg) => process.argv.slice(2).some((a) => a.indexOf(arg) !== -1))

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '~': srcDir,
    },
  },
  define: {
    __GITHUB_PROJECT_LINK__: JSON.stringify('https://github.com/tarampampam/webhook-tester'),
    __LATEST_RELEASE_LINK__: JSON.stringify('https://github.com/tarampampam/webhook-tester/releases/latest'),
  },
  build: {
    emptyOutDir: true,
    outDir: distDir,
    reportCompressedSize: false,
    assetsInlineLimit: 0, // default: 4096 (4 KiB)
    rollupOptions: {
      input: {
        app: join(rootDir, 'index.html'), // the default entry point
      },
      output: {
        entryFileNames: 'js/[name]-[hash].js',
        chunkFileNames: 'js/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
      },
    },
    sourcemap: isWatchMode,
    minify: true,
  },
  esbuild: {
    legalComments: 'none',
  },
  // @ts-ignore-next-line The `vite` type definitions are not up-to-date
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './vitest.setup.js',
  },
})
