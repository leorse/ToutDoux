/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/setupTests.ts'],
    // Les tests de composants ne doivent jamais atteindre le backend : ils
    // reçoivent leurs données en props ou via un double injecté (§3.10).
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
