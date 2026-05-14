import react from '@vitejs/plugin-react'
import { join, resolve } from 'path'
import { defineConfig } from 'vite'

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
    chunkSizeWarningLimit: 2048,
    rolldownOptions: {
      input: {
        app: join(rootDir, 'index.html'), // the default entry point
      },
      output: {
        entryFileNames: 'js/[name]-[hash].js',
        chunkFileNames: 'js/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
        comments: { legal: false },
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
          '^/api/.*': { ws: true, target: devServerProxyTo },
          '/ready': devServerProxyTo,
          '/healthz': devServerProxyTo,
          '^/[0-9a-f-]{36}.*$': devServerProxyTo, // webhook url's
        }
      : undefined,
    allowedHosts: true,
  },
})
