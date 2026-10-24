import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    // adapter-static + fallback: no Node process at runtime, so this
    // builds and serves on plain CGI/shared hosting -- see
    // docs/decisions/0026-sveltekit-starter-static-adapter.md for why.
    // $lib: SvelteKit 3 removed the built-in alias in favor of #lib; keep
    // $lib working rather than rewrite every import in this starter.
    sveltekit({
      adapter: adapter({ fallback: 'index.html', strict: false }),
      alias: { $lib: 'src/lib' },
    }),
  ],
});
