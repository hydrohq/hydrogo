import type { ProductItem } from './routes/storefront/ProductDetailModal.svelte';

export interface HydroDataPayload {
  store?: {
    name?: string;
    tagline?: string;
    phone?: string;
    currency?: string;
    logo_url?: string;
    announcement?: string;
    primary_color?: string;
    bg_tone?: string;
  };
  products?: ProductItem[];
}

declare global {
  interface Window {
    __HYDRO_DATA__?: HydroDataPayload;
  }
}

export {};
