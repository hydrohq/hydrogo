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

const base = process.env.VITE_ADMIN_BASE || '/admin/';
const isDemo = process.env.VITE_DEMO_MODE === 'true';

export default defineConfig({
  base,
  plugins: [
    svelte(),
    tailwindcss(),
    ...(isDemo ? [] : [copyToAdminUi()]),
  ],
  resolve: {
    alias: {
      $lib: path.resolve(__dirname, './src/lib'),
    },
    conditions: process.env.VITEST ? ['browser'] : undefined,
  },
  test: {
    // Default stays Node; component tests opt in with a
    // `// @vitest-environment jsdom` comment at the top of the file.
    environment: 'node',
    setupFiles: ['./vitest-setup.ts'],
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
    rollupOptions: {
      input: {
        index: path.resolve(__dirname, 'index.html'),
        ...(fs.existsSync(path.resolve(__dirname, 'storefront.html'))
          ? { storefront: path.resolve(__dirname, 'storefront.html') }
          : {}),
      },
    },
  },
});
