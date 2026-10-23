import { error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { hydro, toHttpError } from '$lib/server/hydro';
import { addToCart, getStock } from '$lib/server/commerce';
import type { Product, ProductImage, Variant } from '$lib/catalog';

const RELATED_LIMIT = 4;

export const load: PageServerLoad = async ({ params }) => {
  try {
    const products = await hydro.listContent<Product>('products', { limit: 1, expand: ['category'], filter: { slug: params.slug } });
    const product = products.data[0];
    if (!product) error(404, 'Ürün bulunamadı');

    const [variants, gallery, related] = await Promise.all([
      hydro.listContent<Variant>('variants', { limit: 100, filter: { product: product.id } }),
      hydro.listContent<ProductImage>('product_images', { limit: 20, sort: 'sort_order', filter: { product: product.id } }),
      product.category
        ? hydro.listContent<Product>('products', { limit: RELATED_LIMIT + 1, sort: '-created_at', filter: { category: product.category } })
        : Promise.resolve({ data: [] as Product[], metadata: { limit: 0, has_more: false } })
    ]);
    const stockByVariant = Object.fromEntries(await Promise.all(variants.data.map(async (variant) => [variant.id, await getStock(variant.id)] as const)));
    const relatedProducts = related.data.filter((candidate) => candidate.id !== product.id).slice(0, RELATED_LIMIT);

    return { product, variants: variants.data, stockByVariant, gallery: gallery.data, relatedProducts };
  } catch (err) {
    toHttpError(err);
  }
};

export const actions: Actions = {
  addToCart: async ({ request, cookies }) => {
    const form = await request.formData();
    const variantId = String(form.get('variant_id') ?? '').trim();
    const quantity = Number(form.get('quantity') ?? 1);
    const unitPriceCents = Number(form.get('unit_price_cents') ?? 0);
    if (!variantId || !Number.isInteger(quantity) || quantity < 1 || quantity > 20) {
      return fail(400, { message: 'Lütfen bir seçenek ve geçerli bir adet belirtin.' });
    }
    try {
      await addToCart(cookies, variantId, quantity, unitPriceCents);
    } catch {
      return fail(502, { message: 'Ürün sepete eklenemedi, lütfen tekrar deneyin.' });
    }
    return { added: true };
  }
};
