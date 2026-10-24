import { hydro } from '$lib/server/hydro';
import type { Category } from '$lib/catalog';

/** Resolves a category's record id from its slug -- product.category stores
 * the id, not the slug, so a /kategori/[slug] route needs this lookup
 * before it can filter products. */
export async function findCategoryBySlug(slug: string): Promise<Category | null> {
  const match = await hydro.listContent<Category>('categories', { limit: 1, filter: { slug } });
  return match.data[0] ?? null;
}
