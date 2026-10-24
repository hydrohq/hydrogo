import type { LayoutServerLoad } from './$types';
import { hydro } from '$lib/server/hydro';
import { getCart, getCurrencies } from '$lib/server/commerce';
import { currentMember } from '$lib/server/member';
import type { Category } from '$lib/catalog';

const DISPLAY_CURRENCY_COOKIE = 'display_currency';

export const load: LayoutServerLoad = async ({ cookies }) => {
  // Warms the cart + CSRF cookies for this visitor before any page's form
  // action tries to mutate the cart -- see commerce.ts's own doc comment on
  // why this has to happen before the first write, not just at checkout.
  const [cart, categories, currencies, member] = await Promise.all([
    getCart(cookies).catch(() => ({ id: '', items: [] })),
    hydro.listContent<Category>('categories', { limit: 50, sort: 'name' }).catch(() => ({ data: [] as Category[], metadata: { limit: 50, has_more: false } })),
    getCurrencies(),
    currentMember(cookies)
  ]);
  const itemCount = cart.items.reduce((total, item) => total + item.quantity, 0);

  const requested = cookies.get(DISPLAY_CURRENCY_COOKIE) ?? '';
  const displayCurrency = requested === currencies.default || currencies.display.includes(requested) ? requested : currencies.default;

  return { cartItemCount: itemCount, categories: categories.data, currencies, displayCurrency, member };
};
