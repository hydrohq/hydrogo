export type Block = { type: string; payload: Record<string, unknown> };
export type Page = { id: string; locale: string; slug: string; title: string; body: string; blocks: Block[]; status: "draft" | "published"; revision: number; published_revision?: number };
export type NewsItem = { id: string; locale: string; slug: string; title: string; body: string; status: "published"; revision: number; published_revision: number; updated_at: string };
export type Product = { id: string; sku: string; title: string; price_cents: number; currency: string; stock: number; status: "draft" | "published" | "out_of_stock" };
export type ProductVariant = Product & { product_id: string };
export type CartItem = { id: string; product_id: string; variant_id?: string; sku: string; title: string; quantity: number; unit_cents: number; total_cents: number };
export type Cart = { id: string; currency: string; items: CartItem[]; total_cents: number };
export type CheckoutInput = { cart_id?: string; items?: Array<{ product_id: string; variant_id?: string; quantity: number }>; customer: { name: string; identity_number: string; email: string; phone: string; address: string; city: string; country: string; postal_code?: string } };
export type Order = { id: string; currency: string; total_cents: number; payment_status: string; shipping_status: string };

export class HydroGoError extends Error {
  constructor(readonly status: number, readonly body: string) { super(`HydroGo API returned ${status}`); }
}

export type HydroGoClientOptions = { origin: string; fetch?: typeof fetch };

export class HydroGoClient {
  private readonly origin: string;
  private readonly requestFetch: typeof fetch;

  constructor(options: HydroGoClientOptions) {
    this.origin = options.origin.replace(/\/$/, "");
    this.requestFetch = options.fetch ?? fetch;
  }

  async getPage(locale: string, slug: string): Promise<Page> {
    return this.request<Page>(`/api/v1/pages/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`);
  }

  async listPages(locale: string, limit = 20): Promise<Page[]> {
    if (!Number.isInteger(limit) || limit < 1 || limit > 100) throw new RangeError("limit must be between 1 and 100");
    return this.request<Page[]>(`/api/v1/pages?locale=${encodeURIComponent(locale)}&limit=${limit}`);
  }

  async listNews(locale: string, limit = 20): Promise<NewsItem[]> {
    if (!Number.isInteger(limit) || limit < 1 || limit > 100) throw new RangeError("limit must be between 1 and 100");
    return this.request<NewsItem[]>(`/api/v1/news?locale=${encodeURIComponent(locale)}&limit=${limit}`);
  }

  async getNews(locale: string, slug: string): Promise<NewsItem> {
    return this.request<NewsItem>(`/api/v1/news/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`);
  }

  async listProducts(): Promise<Product[]> { return this.request<Product[]>("/api/v1/commerce/products"); }

  async listVariants(productID: string): Promise<ProductVariant[]> {
    return this.request<ProductVariant[]>(`/api/v1/commerce/products/${encodeURIComponent(productID)}/variants`);
  }

  async getCart(): Promise<Cart> { return this.request<Cart>("/api/v1/commerce/cart", { credentials: "include" }); }

  async addCartItem(input: { product_id: string; variant_id?: string; quantity: number }, csrfToken: string): Promise<Cart> {
    return this.request<Cart>("/api/v1/commerce/cart/items", this.mutation(input, csrfToken));
  }

  async removeCartItem(productID: string, csrfToken: string): Promise<void> {
    await this.request<void>(`/api/v1/commerce/cart/items/${encodeURIComponent(productID)}`, { method: "DELETE", credentials: "include", headers: { "X-CSRF-Token": csrfToken } });
  }

  async checkout(input: CheckoutInput, idempotencyKey: string, csrfToken: string): Promise<Order> {
    return this.request<Order>("/api/v1/commerce/cart/checkout", this.mutation(input, csrfToken, { "Idempotency-Key": idempotencyKey }));
  }

  private mutation(body: unknown, csrfToken: string, extraHeaders: Record<string, string> = {}): RequestInit {
    return { method: "POST", credentials: "include", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken, ...extraHeaders }, body: JSON.stringify(body) };
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await this.requestFetch(`${this.origin}${path}`, init);
    if (response.status === 204) return undefined as T;
    if (!response.ok) throw new HydroGoError(response.status, await response.text());
    return response.json() as Promise<T>;
  }
}

export function createHydroGoClient(options: HydroGoClientOptions): HydroGoClient { return new HydroGoClient(options); }
