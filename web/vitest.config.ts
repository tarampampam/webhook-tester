import viteConfig from './vite.config'
import { mergeConfig } from 'vite'
import { defineConfig } from 'vitest/config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      setupFiles: './vitest/setup.ts',
      globalSetup: './vitest/setup.global.ts',
      projects: [
        {
          extends: true,
          test: {
            name: 'dom ',
            include: ['src/**/*.tsx.test.*', 'src/**/*.test.tsx'],
            environment: 'happy-dom',
          },
        },
        {
          extends: true,
          test: {
            name: 'node',
            include: ['src/**/*.ts.test.*', 'src/**/*.test.ts'],
            environment: 'node',
          },
        },
      ],
      coverage: {
        exclude: ['**/*.gen.ts', '**/*.gen.js'],
      },
    },
  })
)
