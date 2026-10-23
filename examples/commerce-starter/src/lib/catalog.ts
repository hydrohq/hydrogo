// Shapes of content-types/commerce/*.yaml's records, as the public content
// API returns them. Stock is deliberately not here -- it lives in
// modules/commerce's own table, read through commerce.ts's getStock.

export type Category = {
  id: string;
  name: string;
  slug: string;
  parent?: string;
  description?: string;
};

export type Product = {
  id: string;
  name: string;
  slug: string;
  category?: string;
  description?: string;
  cover_image: string;
  base_price: number;
  currency: string;
  sku?: string;
  seo_title?: string;
  seo_description?: string;
  expanded?: { category?: Category };
};

export type Variant = {
  id: string;
  product: string;
  sku: string;
  size?: string;
  color?: string;
  price_override?: number;
  expanded?: { product?: Product };
};

/** A variant's effective price: its own override, or the product's base price. */
export function variantPriceCents(variant: Variant, product: Product): number {
  const price = variant.price_override ?? product.base_price;
  return Math.round(price * 100);
}

export function formatPrice(cents: number, currency: string): string {
  const amount = (cents / 100).toLocaleString('tr-TR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  const symbol = currency === 'TRY' ? '₺' : currency === 'USD' ? '$' : currency === 'EUR' ? '€' : currency + ' ';
  return currency === 'TRY' ? `${amount} ${symbol}` : `${symbol}${amount}`;
}
