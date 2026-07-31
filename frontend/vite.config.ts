import { defineConfig } from 'vite-plus'
import vue from '@vitejs/plugin-vue'
import ui from '@tabtab/ui/vite'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webOutDir = resolve(__dirname, '../web/dist')

export default defineConfig({
  // 使用相对资源路径，兼容 Nginx 根路径和 /admin/ 等子路径部署。
  base: '/',

  plugins: [vue(), ui()],

  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },

  server: {
    port: 3001,
    host: true,
  },

  build: {
    outDir: webOutDir,
    emptyOutDir: true,
  },

  test: {
    globals: true,
    include: ['**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
    exclude: ['**/node_modules/**', '**/dist/**', '**/web/**'],
  },

  fmt: {
    semi: false,
    singleQuote: true,
  },

  run: {
    tasks: {
      buildAll: {
        command: 'vite build',
      },
    },
  },
})
