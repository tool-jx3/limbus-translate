import preact from '@preact/preset-vite'
import { defineConfig } from 'vitest/config'

// The site is served under /<repo>/, so every URL in the build is relative.
export default defineConfig({
  base: './',
  plugins: [preact()],
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
