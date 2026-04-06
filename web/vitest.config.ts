import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue()],
  esbuild: {
    tsconfigRaw: {
      compilerOptions: {
        module: 'ESNext',
        moduleResolution: 'Bundler',
        target: 'ES2022',
      },
    },
  },
  resolve: {
    alias: {
      '~': fileURLToPath(new URL('./app', import.meta.url)),
      '@': fileURLToPath(new URL('./app', import.meta.url)),
    },
  },
  test: {
    environment: 'happy-dom',
    setupFiles: ['./test/setup.ts'],
  },
})
