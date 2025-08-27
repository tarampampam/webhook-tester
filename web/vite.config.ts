/// <reference types="vite/client" />

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve, join } from 'path'

const rootDir = resolve(__dirname)
const [distDir, srcDir] = [join(rootDir, 'dist'), join(rootDir, 'src')]
const isWatchMode = ['serve', 'dev', 'watch'].some((arg) => process.argv.slice(2).some((a) => a.indexOf(arg) !== -1))
const devServerProxyTo = process.env?.['DEV_SERVER_PROXY_TO'] || undefined

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '~': srcDir,
      // /esm/icons/index.mjs only exports the icons statically, so no separate chunks are created.
      // without this workaround vite dev server sends a bunch of chunks (more than 5k+) to the browser
      // @link https://github.com/tabler/tabler-icons/issues/1233#issuecomment-2428245119
      '@tabler/icons-react': '@tabler/icons-react/dist/esm/icons/index.mjs',
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
  server: {
    strictPort: true,
    open: false,
    proxy: devServerProxyTo
      ? {
          '^/api/.*': devServerProxyTo,
          '^/api/.*/subscribe$': { ws: true, target: devServerProxyTo },
          '/ready': devServerProxyTo,
          '/healthz': devServerProxyTo,
          '^/[0-9a-f-]{36}.*$': devServerProxyTo, // webhook url's
        }
      : undefined,
  },
  esbuild: {
    legalComments: 'none',
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './vitest.setup.js',
  },
})
