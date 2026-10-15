import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

function copyToAdminUi() {
  return {
    name: 'copy-to-adminui',
    closeBundle() {
      const srcDir = path.resolve(__dirname, 'dist');
      const destDir = path.resolve(__dirname, '../internal/adminui/dist');
      if (fs.existsSync(srcDir)) {
        if (fs.existsSync(destDir)) {
          fs.rmSync(destDir, { recursive: true, force: true });
        }
        fs.mkdirSync(destDir, { recursive: true });
        fs.cpSync(srcDir, destDir, { recursive: true, force: true });
      }
    },
  };
}

export default defineConfig({
  base: '/admin/',
  plugins: [
    svelte(),
    tailwindcss(),
    copyToAdminUi(),
  ],
  resolve: {
    alias: {
      $lib: path.resolve(__dirname, './src/lib'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
      '/csrf-token': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
