import type { Cookies } from '@sveltejs/kit';
import { hydro } from '$lib/server/hydro';
import { getCart } from '$lib/server/commerce';
import type { Product, Variant } from '$lib/catalog';

export type CartLine = {
  itemId: string;
  variantId: string;
  quantity: number;
  unitPriceCents: number;
  totalCents: number;
  productName: string;
  productSlug: string;
  productImage: string;
  productSKU: string;
  variantLabel: string;
};

async function describeLine(variantId: string, quantity: number, unitPriceCents: number, itemId: string): Promise<CartLine | null> {
  try {
    const variant = await hydro.getContent<Variant>('variants', variantId);
    const product = await hydro.getContent<Product>('products', variant.product);
    return {
      itemId,
      variantId,
      quantity,
      unitPriceCents,
      totalCents: unitPriceCents * quantity,
      productName: product.name,
      productSlug: product.slug,
      productImage: product.cover_image,
      productSKU: variant.sku || product.sku || '',
      variantLabel: [variant.size, variant.color].filter(Boolean).join(' / ')
    };
  } catch {
    return null;
  }
}

/** variant_id -> title/sku maps, exactly what checkout()'s item_titles/
 * item_skus expect -- commerce has no access to the catalog records that
 * own this data, so whoever already resolved it for display (here) is
 * who supplies it at checkout time. */
export function itemCatalogData(lines: CartLine[]): { itemTitles: Record<string, string>; itemSKUs: Record<string, string> } {
  const itemTitles: Record<string, string> = {};
  const itemSKUs: Record<string, string> = {};
  for (const line of lines) {
    const label = line.variantLabel ? `${line.productName} (${line.variantLabel})` : line.productName;
    itemTitles[line.variantId] = label;
    itemSKUs[line.variantId] = line.productSKU;
  }
  return { itemTitles, itemSKUs };
}

/** The signed-in visitor's cart, with each line's catalog details resolved
 * for display -- the cart API itself only carries variant_id/quantity/price,
 * it has no idea about product names or images (that is content-type data,
 * not commerce's). */
export async function loadCartLines(cookies: Cookies): Promise<{ lines: CartLine[]; totalCents: number }> {
  // A commerce API outage shows an empty cart, not a 500: the shopper can
  // still browse, and the alternative (crashing the page) is worse than
  // momentarily looking empty -- same choice the root layout's own cart
  // warm-up already makes.
  const cart = await getCart(cookies).catch(() => ({ id: '', items: [] }));
  const lines = (await Promise.all(cart.items.map((item) => describeLine(item.variant_id, item.quantity, item.unit_price_cents, item.id)))).filter(
    (line): line is CartLine => line !== null
  );
  const totalCents = lines.reduce((total, line) => total + line.totalCents, 0);
  return { lines, totalCents };
}
