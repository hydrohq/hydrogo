import type { PageServerLoad } from './$types';
import { hydro, toHttpError } from '$lib/server/hydro';
import { getCheckoutConfig } from '$lib/server/commerce';
import type { Category, Product } from '$lib/catalog';

const PRODUCTS_PER_SECTION = 4;

export type CategorySection = { category: Category; products: Product[] };

export const load: PageServerLoad = async () => {
  try {
    const [categories, checkoutConfig] = await Promise.all([hydro.listContent<Category>('categories', { limit: 6, sort: 'name' }), getCheckoutConfig()]);

    const sections: CategorySection[] = [];
    for (const category of categories.data) {
      const products = await hydro.listContent<Product>('products', { limit: PRODUCTS_PER_SECTION, sort: '-created_at', filter: { category: category.id } });
      if (products.data.length > 0) sections.push({ category, products: products.data });
    }

    // No categories (or none with products) yet -- fall back to a flat,
    // uncategorised grid of the most recent products, so an empty/fresh
    // catalog still shows something instead of nothing.
    let latest: Product[] = [];
    if (sections.length === 0) {
      const products = await hydro.listContent<Product>('products', { limit: 24, sort: '-created_at', expand: ['category'] });
      latest = products.data;
    }

    return { sections, latest, freeShippingAboveCents: checkoutConfig.shipping_free_above_cents };
  } catch (err) {
    toHttpError(err);
  }
};
