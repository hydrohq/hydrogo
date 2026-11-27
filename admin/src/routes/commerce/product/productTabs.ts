// Extension slot for the product editor: any `*.svelte` file dropped into
// `./extensions/` that exports `tab` from a `<script module>` block (and has a
// default component taking `{ product, onsaved }`) becomes an extra tab. The
// directory is empty in core; deployments add their own tabs by overlaying files
// into it before building the admin.
import type { Component } from 'svelte';
import type { ProductRecord } from '../ProductEditorView.svelte';

export interface ProductTabContext {
  product: ProductRecord | null;
  isNew: boolean;
  /** True when the product's category has this slug (resolved from id or object). */
  isInCategory: (slug: string) => boolean;
}

export interface ProductTabDescriptor {
  id: string;
  label: string | (() => string);
  visible?: (ctx: ProductTabContext) => boolean;
}

export interface ProductTab extends ProductTabDescriptor {
  component: Component<{ product: ProductRecord | null; onsaved?: (updated: ProductRecord) => void }>;
}

const modules = import.meta.glob<{ default: ProductTab['component']; tab?: ProductTabDescriptor }>(
  './extensions/*.svelte',
  { eager: true }
);

export const productTabs: ProductTab[] = Object.values(modules)
  .filter((m) => m.tab?.id && m.default)
  .map((m) => ({ ...(m.tab as ProductTabDescriptor), component: m.default }));
