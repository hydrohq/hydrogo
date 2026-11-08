import { mount } from 'svelte';
import './app.css';
import StorefrontApp from './routes/storefront/StorefrontApp.svelte';

const target = document.getElementById('storefront-app');
if (target) {
  // Clear any SSR placeholder content before mounting interactive Svelte app
  target.innerHTML = '';
  mount(StorefrontApp, { target });
}
