import { defineConfig } from 'tsdown'
import Vue from 'unplugin-vue/rolldown'

export default defineConfig({
  entry: {
    index: './src/index.ts',
    vite: './src/vite/index.ts',
  },
  format: ['esm'],
  platform: 'node',
  deps: {
    neverBundle: [
      'vue',
      'vue-router',
      '@vueuse/core',
      'tailwindcss',
      'class-variance-authority',
      'clsx',
      'tailwind-merge',
      '@lucide/vue',
      'reka-ui',
      '@tanstack/vue-table',
      '@tanstack/vue-virtual',
      '@unovis/vue',
      '@internationalized/date',
      'embla-carousel',
      'embla-carousel-vue',
      'vaul-vue',
      'vee-validate',
      '@vee-validate/zod',
      'vue-input-otp',
      'vue-sonner',
      'zod',
      'tw-animate-css',
      '@tabtab/utils',
      'vite',
    ],
  },
  plugins: [
    Vue({
      isProduction: true,
      script: {
        defineModel: true,
        propsDestructure: true,
      },
    }),
  ],
  dts: {
    vue: true,
  },
  clean: true,
})
