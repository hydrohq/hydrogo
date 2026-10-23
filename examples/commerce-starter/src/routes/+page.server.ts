import type { PageServerLoad } from './$types';
import { hydro, toHttpError } from '$lib/server/hydro';
import type { Category, Product } from '$lib/catalog';

export const load: PageServerLoad = async ({ url }) => {
  const categorySlug = url.searchParams.get('kategori') ?? '';
  try {
    let categoryId = '';
    if (categorySlug) {
      const match = await hydro.listContent<Category>('categories', { limit: 1, filter: { slug: categorySlug } });
      categoryId = match.data[0]?.id ?? '';
    }
    const products = await hydro.listContent<Product>('products', {
      limit: 48,
      sort: '-created_at',
      expand: ['category'],
      ...(categoryId ? { filter: { category: categoryId } } : {})
    });
    return { products: products.data, activeCategorySlug: categorySlug };
  } catch (err) {
    toHttpError(err);
  }
};
