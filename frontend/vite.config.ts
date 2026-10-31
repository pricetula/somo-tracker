import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [
    tanstackStart(),
    tailwindcss(),
    react(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: true,
    port: 3000,
    proxy: {
      '/api/v1': {
        target: process.env.PUBLIC_API_URL || process.env.BACKEND_URL || 'http://localhost:8080',
        changeOrigin: true,
        timeout: 300_000, // 5 minutes for CSV/PDF downloads
        proxyTimeout: 300_000,
      },
    },
  },
});