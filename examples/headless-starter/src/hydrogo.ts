export type Block = { type: string; payload: Record<string, unknown> };
export type Page = { id: string; locale: string; slug: string; title: string; body: string; blocks: Block[]; status: "draft" | "published"; revision: number; published_revision?: number };
export type PageSummary = { id: string; locale: string; slug: string; title: string; revision: number; publication_generation: number; updated_at: string };
export type NewsItem = { id: string; locale: string; slug: string; title: string; body: string; status: "published"; revision: number; published_revision: number; updated_at: string };
export type Product = { id: string; sku: string; title: string; price_cents: number; currency: string; stock: number; status: "draft" | "published" | "out_of_stock" };
export type ProductVariant = Product & { product_id: string };
export type CartItem = { id: string; product_id: string; variant_id?: string; sku: string; title: string; quantity: number; unit_cents: number; total_cents: number };
export type Cart = { id: string; currency: string; items: CartItem[]; total_cents: number };
export type CheckoutInput = { cart_id?: string; items?: Array<{ product_id: string; variant_id?: string; quantity: number }>; customer: { name: string; identity_number: string; email: string; phone: string; address: string; city: string; country: string; postal_code?: string } };
export type Order = { id: string; currency: string; total_cents: number; payment_status: string; shipping_status: string };

export type HydroAPIErrorBody = { code?: string; message?: string; error?: string; request_id?: string };

export class HydroGoError extends Error {
  readonly code: string;
  readonly requestId: string;
  constructor(readonly status: number, readonly body?: HydroAPIErrorBody, requestId = "") {
    super(body?.message ?? body?.error ?? `HydroGo API returned ${status}`);
    this.name = "HydroGoError";
    this.code = body?.code ?? "api_error";
    this.requestId = body?.request_id ?? requestId;
  }
}

export function validateIdempotencyKey(key: string): string {
  if (!/^[\x21-\x7e]{16,128}$/.test(key)) throw new RangeError("Idempotency-Key must be 16–128 visible ASCII characters");
  return key;
}

export type HydroGoClientOptions = { origin: string; fetch?: typeof fetch; timeoutMs?: number; signal?: AbortSignal };

const DEFAULT_TIMEOUT_MS = 15_000;
const MAX_TIMEOUT_MS = 120_000;
const MAX_ERROR_BODY_CODEPOINTS = 8_192;
const MAX_RESPONSE_BYTES = 2 * 1024 * 1024;

async function readBoundedResponse(response: Response, maxBytes = MAX_RESPONSE_BYTES): Promise<string> {
  const reader = response.body?.getReader();
  if (!reader) {
    const text = await response.text();
    if (new TextEncoder().encode(text).byteLength > maxBytes) throw new HydroGoError(response.status, { code: "response_too_large", message: "HydroGo API response is too large" }, response.headers.get("x-request-id") ?? "");
    return text;
  }
  const decoder = new TextDecoder();
  let bytes = 0;
  let text = "";
  try {
    while (true) {
      const chunk = await reader.read();
      if (chunk.done) break;
      bytes += chunk.value.byteLength;
      if (bytes > maxBytes) {
        await reader.cancel();
        throw new HydroGoError(response.status, { code: "response_too_large", message: "HydroGo API response is too large" }, response.headers.get("x-request-id") ?? "");
      }
      text += decoder.decode(chunk.value, { stream: true });
    }
    return text + decoder.decode();
  } finally {
    reader.releaseLock();
  }
}

export class HydroGoClient {
  private readonly origin: string;
  private readonly requestFetch: typeof fetch;
  private readonly timeoutMs: number;
  private readonly externalSignal?: AbortSignal;

  constructor(options: HydroGoClientOptions) {
    this.origin = options.origin.replace(/\/$/, "");
    this.requestFetch = options.fetch ?? fetch;
    const timeout = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    if (!Number.isInteger(timeout) || timeout < 1 || timeout > MAX_TIMEOUT_MS) throw new RangeError("timeoutMs must be between 1 and 120000");
    this.timeoutMs = timeout;
    this.externalSignal = options.signal;
  }

  async getPage(locale: string, slug: string): Promise<Page> {
    return this.request<Page>(`/api/v1/pages/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`);
  }

  async listPages(locale: string, limit = 20): Promise<PageSummary[]> {
    if (!Number.isInteger(limit) || limit < 1 || limit > 100) throw new RangeError("limit must be between 1 and 100");
    return this.request<PageSummary[]>(`/api/v1/pages?locale=${encodeURIComponent(locale)}&limit=${limit}`);
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
    return this.request<Order>("/api/v1/commerce/cart/checkout", this.mutation(input, csrfToken, { "Idempotency-Key": validateIdempotencyKey(idempotencyKey) }));
  }

  private mutation(body: unknown, csrfToken: string, extraHeaders: Record<string, string> = {}): RequestInit {
    return { method: "POST", credentials: "include", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken, ...extraHeaders }, body: JSON.stringify(body) };
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    const externalSignal = init.signal ?? this.externalSignal;
    const forwardAbort = () => controller.abort(externalSignal?.reason);
    if (externalSignal) {
      if (externalSignal.aborted) forwardAbort();
      else externalSignal.addEventListener("abort", forwardAbort, { once: true });
    }
    try {
      const response = await this.requestFetch(`${this.origin}${path}`, { ...init, signal: controller.signal });
      const declaredLength = Number(response.headers.get("content-length") || 0);
      if (declaredLength > 2 * 1024 * 1024) {
        throw new HydroGoError(response.status, { code: "response_too_large", message: "HydroGo API response is too large" }, response.headers.get("x-request-id") ?? "");
      }
      if (response.status === 204) return undefined as T;
      if (!response.ok) {
        const raw = [...(await readBoundedResponse(response))].slice(0, MAX_ERROR_BODY_CODEPOINTS).join("");
        let body: HydroAPIErrorBody | undefined;
        try { body = JSON.parse(raw) as HydroAPIErrorBody; } catch { /* keep body absent */ }
        throw new HydroGoError(response.status, body, response.headers.get("x-request-id") ?? "");
      }
      const raw = await readBoundedResponse(response);
      return JSON.parse(raw) as T;
    } finally {
      clearTimeout(timer);
      externalSignal?.removeEventListener("abort", forwardAbort);
    }
  }
}

export function createHydroGoClient(options: HydroGoClientOptions): HydroGoClient { return new HydroGoClient(options); }
