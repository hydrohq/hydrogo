import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';
import fs from 'fs';

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
    tailwindcss(),
    svelte(),
    copyToAdminUi(),
  ],
  resolve: {
    alias: {
      $lib: path.resolve(__dirname, './src/lib'),
    },
  },
  server: {
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
