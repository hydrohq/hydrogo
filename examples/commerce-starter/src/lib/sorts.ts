// Plain data, safe for client code: the component that renders the sort
// links needs these labels/keys too, so this cannot live in $lib/server.

export const PRODUCT_SORTS = {
  newest: { label: 'En yeni', query: '-created_at' },
  price_asc: { label: 'Fiyat: Artan', query: 'base_price' },
  price_desc: { label: 'Fiyat: Azalan', query: '-base_price' },
  name: { label: 'İsme göre', query: 'name' }
} as const;

export type ProductSortKey = keyof typeof PRODUCT_SORTS;

export function resolveSort(value: string | null): ProductSortKey {
  return value && value in PRODUCT_SORTS ? (value as ProductSortKey) : 'newest';
}
