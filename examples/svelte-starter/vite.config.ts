import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [
    tailwindcss(),
    svelte(),
  ],
  resolve: {
    alias: {
      $lib: path.resolve(__dirname, './src/lib'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.HYDRO_API_ORIGIN || 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
      '/media': {
        target: process.env.HYDRO_API_ORIGIN || 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
    },
  },
});
