import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { addToCart, removeFromCart } from '$lib/server/commerce';
import { loadCartLines } from '$lib/server/cartLines';

export const load: PageServerLoad = async ({ cookies }) => {
  const { lines, totalCents } = await loadCartLines(cookies);
  return { lines, totalCents, currency: 'TRY' };
};

export const actions: Actions = {
  updateQuantity: async ({ request, cookies }) => {
    const form = await request.formData();
    const variantId = String(form.get('variant_id') ?? '');
    const unitPriceCents = Number(form.get('unit_price_cents') ?? 0);
    const quantity = Number(form.get('quantity') ?? 1);
    if (!variantId || !Number.isInteger(quantity) || quantity < 1 || quantity > 20) {
      return fail(400, { message: 'Geçersiz adet.' });
    }
    try {
      await addToCart(cookies, variantId, quantity, unitPriceCents);
    } catch {
      return fail(502, { message: 'Sepet güncellenemedi.' });
    }
    return { updated: true };
  },
  remove: async ({ request, cookies }) => {
    const form = await request.formData();
    const itemId = String(form.get('item_id') ?? '');
    if (!itemId) return fail(400, { message: 'Geçersiz ürün.' });
    try {
      await removeFromCart(cookies, itemId);
    } catch {
      return fail(502, { message: 'Ürün sepetten çıkarılamadı.' });
    }
    return { removed: true };
  }
};
