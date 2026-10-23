import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  // adapter-node: every request is rendered on the server, so a stock change,
  // a price edit or a new product in the HydroGo admin is live immediately --
  // no rebuild, no cache to bust (AGENTS.md rule 12, zero build on content
  // publish).
  kit: { adapter: adapter() }
};
