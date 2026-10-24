import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { hydro, toHttpError } from '$lib/server/hydro';
import { findCategoryBySlug } from '$lib/server/categories';
import { PRODUCT_SORTS, resolveSort } from '$lib/sorts';
import type { Product } from '$lib/catalog';

export const load: PageServerLoad = async ({ params, url }) => {
  try {
    const category = await findCategoryBySlug(params.slug);
    if (!category) error(404, 'Kategori bulunamadı');

    const sortKey = resolveSort(url.searchParams.get('sort'));
    const products = await hydro.listContent<Product>('products', {
      limit: 48,
      sort: PRODUCT_SORTS[sortKey].query,
      expand: ['category'],
      filter: { category: category.id }
    });

    return { category, products: products.data, sortKey };
  } catch (err) {
    toHttpError(err);
  }
};
