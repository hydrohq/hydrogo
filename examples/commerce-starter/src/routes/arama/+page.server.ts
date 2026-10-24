import type { PageServerLoad } from './$types';
import { hydro, toHttpError } from '$lib/server/hydro';
import type { Product } from '$lib/catalog';

export const load: PageServerLoad = async ({ url }) => {
  const query = (url.searchParams.get('q') ?? '').trim();
  if (!query) return { query, products: [] as Product[] };
  try {
    const products = await hydro.listContent<Product>('products', { limit: 48, search: query, expand: ['category'] });
    return { query, products: products.data };
  } catch (err) {
    toHttpError(err);
  }
};
