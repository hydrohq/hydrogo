import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  // adapter-node: pages are rendered on the server for every request, so a
  // publish in the HydroGo admin is live immediately and crawlers and link
  // previews receive complete HTML. No rebuild is needed when content changes.
  kit: { adapter: adapter() }
};
