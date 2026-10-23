import type { Cookies } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';
import type { CurrenciesConfig } from '$lib/catalog';

const API_ORIGIN = (env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
export const SECURE_COOKIES = publicEnv.PUBLIC_SECURE_COOKIES === 'true';

// The only cookies this storefront ever needs to carry between the browser
// and HydroGo. Named exactly as HydroGo names them, so a value read from one
// of HydroGo's Set-Cookie headers can be replayed verbatim as this site's
// own cookie of the same name, and forwarded back to HydroGo unchanged on
// the next request -- a proxy, not a session store: there is no privileged
// secret here to keep off the browser (contrast with examples/crm-starter's
// staff session, which does need that indirection).
const PROXIED_COOKIES = ['hydro_commerce_cart', 'hydro_csrf', 'hydro_member_session'] as const;

export class CommerceAPIError extends Error {
  readonly code: string;
  readonly requestId: string;
  constructor(readonly status: number, body: { code?: string; message?: string; request_id?: string } | null) {
    super(body?.message ?? `HydroGo commerce API returned ${status}`);
    this.name = 'CommerceAPIError';
    this.code = body?.code ?? 'api_error';
    this.requestId = body?.request_id ?? '';
  }
}

function cookieHeader(cookies: Cookies): string {
  return PROXIED_COOKIES.map((name) => [name, cookies.get(name)] as const)
    .filter(([, value]) => value)
    .map(([name, value]) => `${name}=${value}`)
    .join('; ');
}

function applySetCookies(response: Response, cookies: Cookies): void {
  const raw = typeof response.headers.getSetCookie === 'function' ? response.headers.getSetCookie() : [];
  for (const entry of raw) {
    const [pair] = entry.split(';');
    const eq = pair.indexOf('=');
    if (eq < 1) continue;
    const name = pair.slice(0, eq).trim();
    const value = pair.slice(eq + 1).trim();
    if (!(PROXIED_COOKIES as readonly string[]).includes(name)) continue;
    const cleared = /expires=Thu, 01 Jan 1970/i.test(entry) || /max-age=0/i.test(entry) || value === '';
    if (cleared) {
      cookies.delete(name, { path: '/' });
      continue;
    }
    // HttpOnly regardless of how HydroGo set it: nothing here is read by
    // this starter's own client-side JS, every mutation is a server form
    // action, so there is no reason to ever expose these to the browser's
    // script context.
    cookies.set(name, value, { path: '/', httpOnly: true, sameSite: 'lax', secure: SECURE_COOKIES, maxAge: 60 * 60 * 24 * 30 });
  }
}

async function commerceRequest<T>(cookies: Cookies, path: string, init: RequestInit = {}): Promise<T> {
  const method = (init.method ?? 'GET').toUpperCase();
  const headers = new Headers(init.headers);
  const forwarded = cookieHeader(cookies);
  if (forwarded) headers.set('Cookie', forwarded);
  if (method !== 'GET' && method !== 'HEAD') {
    const csrf = cookies.get('hydro_csrf');
    if (csrf) headers.set('X-CSRF-Token', csrf);
  }
  const response = await fetch(`${API_ORIGIN}${path}`, { ...init, method, headers });
  applySetCookies(response, cookies);
  const text = await response.text();
  const parsed = text ? JSON.parse(text) : null;
  if (!response.ok) throw new CommerceAPIError(response.status, parsed);
  return (parsed?.data ?? parsed) as T;
}

export type CartItem = { id: string; variant_id: string; quantity: number; unit_price_cents: number };
export type Cart = { id: string; items: CartItem[] };

export type Order = {
  id: string;
  member_id?: string;
  guest_name?: string;
  guest_email?: string;
  currency: string;
  subtotal_cents: number;
  discount_code?: string;
  discount_cents: number;
  tax_cents: number;
  shipping_cents: number;
  total_cents: number;
  payment_status: 'pending' | 'paid' | 'failed' | 'refunded';
  fulfillment_status: string;
  created_at: string;
  items?: { variant_id: string; sku: string; title: string; quantity: number; unit_price_cents: number; total_cents: number }[];
};

export type PaymentStart = { token?: string; redirect_url?: string };
export type CheckoutResult = Order & { payment?: PaymentStart };

/** Ensures a cart (and the CSRF cookie every write needs) exists for this
 * visitor before any form action tries to mutate it -- call once per
 * request, from the root layout load. */
export async function getCart(cookies: Cookies): Promise<Cart> {
  return commerceRequest<Cart>(cookies, '/api/v1/commerce/cart');
}

export async function addToCart(cookies: Cookies, variantId: string, quantity: number, unitPriceCents: number): Promise<Cart> {
  return commerceRequest<Cart>(cookies, '/api/v1/commerce/cart/items', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ variant_id: variantId, quantity, unit_price_cents: unitPriceCents })
  });
}

export async function removeFromCart(cookies: Cookies, itemId: string): Promise<Cart> {
  return commerceRequest<Cart>(cookies, `/api/v1/commerce/cart/items/${encodeURIComponent(itemId)}`, { method: 'DELETE' });
}

export type CheckoutInput = {
  idempotencyKey: string;
  guestName: string;
  guestEmail: string;
  guestPhone: string;
  guestAddress: string;
  guestCity: string;
  guestCountry: string;
  guestPostalCode: string;
  discountCode?: string;
  itemTitles?: Record<string, string>;
  itemSKUs?: Record<string, string>;
};

export async function checkout(cookies: Cookies, input: CheckoutInput): Promise<CheckoutResult> {
  return commerceRequest<CheckoutResult>(cookies, '/api/v1/commerce/checkout', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      idempotency_key: input.idempotencyKey,
      guest_name: input.guestName,
      guest_email: input.guestEmail,
      guest_phone: input.guestPhone,
      guest_address: input.guestAddress,
      guest_city: input.guestCity,
      guest_country: input.guestCountry,
      guest_postal_code: input.guestPostalCode,
      discount_code: input.discountCode ?? '',
      item_titles: input.itemTitles ?? {},
      item_skus: input.itemSKUs ?? {}
    })
  });
}

/** Public reads: no cart identity, no CSRF needed (GET), so these do not
 * need the cookie jar -- but keep the same signature shape as the rest of
 * this module for consistency and so a future caller doesn't have to
 * remember which commerce calls are identity-scoped and which are not. */
export async function getStock(variantId: string): Promise<number> {
  const response = await fetch(`${API_ORIGIN}/api/v1/commerce/stock/${encodeURIComponent(variantId)}`);
  if (!response.ok) return 0;
  const body = (await response.json()) as { data?: { available?: number } };
  return body.data?.available ?? 0;
}

export async function getOrder(orderId: string): Promise<Order | null> {
  const response = await fetch(`${API_ORIGIN}/api/v1/commerce/orders/${encodeURIComponent(orderId)}`);
  if (!response.ok) return null;
  const body = (await response.json()) as { data: Order };
  return body.data;
}

export async function startOrderPayment(cookies: Cookies, orderId: string): Promise<PaymentStart> {
  return commerceRequest<PaymentStart>(cookies, `/api/v1/commerce/orders/${encodeURIComponent(orderId)}/payment`, { method: 'POST' });
}

export type CheckoutConfig = { tax_rate: number; tax_inclusive: boolean; shipping_flat_rate_cents: number; shipping_free_above_cents: number };
const DEFAULT_CHECKOUT_CONFIG: CheckoutConfig = { tax_rate: 0, tax_inclusive: false, shipping_flat_rate_cents: 0, shipping_free_above_cents: 0 };

/** Tax/shipping terms, for an upfront estimate before the real checkout
 * (which alone is authoritative) -- see modules/commerce/transport's
 * getCheckoutConfig doc comment. */
// Used from the root layout, so every page depends on this never throwing:
// a commerce API outage must degrade to "no tax, no shipping shown", not
// take the whole site down with it.
export async function getCheckoutConfig(): Promise<CheckoutConfig> {
  try {
    const response = await fetch(`${API_ORIGIN}/api/v1/commerce/checkout-config`);
    if (!response.ok) return DEFAULT_CHECKOUT_CONFIG;
    const body = (await response.json()) as { data?: CheckoutConfig };
    return body.data ?? DEFAULT_CHECKOUT_CONFIG;
  } catch {
    return DEFAULT_CHECKOUT_CONFIG;
  }
}

const DEFAULT_CURRENCIES_CONFIG: CurrenciesConfig = { default: 'TRY', display: [], rates: {} };

// Also used from the root layout -- same reasoning as getCheckoutConfig.
export async function getCurrencies(): Promise<CurrenciesConfig> {
  try {
    const response = await fetch(`${API_ORIGIN}/api/v1/commerce/currencies`);
    if (!response.ok) return DEFAULT_CURRENCIES_CONFIG;
    const body = (await response.json()) as { data?: CurrenciesConfig };
    return body.data ?? DEFAULT_CURRENCIES_CONFIG;
  } catch {
    return DEFAULT_CURRENCIES_CONFIG;
  }
}
