export type Block = { type: string; payload: Record<string, unknown> };
export type Page = { id: string; locale: string; slug: string; title: string; body: string; blocks: Block[]; status: "draft" | "published"; revision: number; published_revision?: number };
export type PageSummary = { id: string; locale: string; slug: string; title: string; revision: number; publication_generation: number; updated_at: string };
export type NewsItem = { id: string; locale: string; slug: string; title: string; body: string; status: "published"; revision: number; published_revision: number; updated_at: string; created_at?: string; excerpt?: string; cover_media_id?: string; category?: string; author_name?: string; tags?: string; seo_title?: string; seo_description?: string };
export type NewsPage = { data: NewsItem[]; metadata: { limit: number; offset?: number; total?: number; has_more: boolean } };
export type MemberTokenPair = { access_token: string; refresh_token: string; token_type: "Bearer"; expires_in: number; expires_at: string };
export type MemberDevice = { device_id: string; device_name: string; platform: string; created_at: string; last_used_at: string; is_current: boolean };
export type MemberProfile = { id: string; email: string; active: boolean; email_verified_at?: string | null; last_login_at?: string | null; created_at: string; updated_at: string };
export type MemberLoginInput = { email: string; password: string; device_id: string; device_name?: string; platform: string };

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

// A filter value is an exact match, or {op: value} with op one of ne, gt, gte, lt, lte, in, null.
export type FilterOp = "ne" | "gt" | "gte" | "lt" | "lte" | "in" | "null";
export type FilterValue = string | Partial<Record<FilterOp, string>>;
// after is the next_cursor of the previous page (keyset paging; use it instead of offset on big lists).
// expand names relation fields whose records come back under "expanded".
export type ContentQuery = { limit?: number; offset?: number; after?: string; sort?: string; search?: string; locale?: string; expand?: string[]; filter?: Record<string, FilterValue> };
export type ContentList<T> = { data: T[]; metadata: { limit: number; has_more: boolean; locale?: string; next_cursor?: string } };
export type HydroGoRequestOptions = { signal?: AbortSignal };
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
    const rawOrigin = options.origin.trim();
    // An empty origin means "same origin as the page" (a relative request) --
    // the common case for a static SPA served from the same domain HydroGo's
    // own API answers on (the usual CGI/static-hosting deployment).
    if (rawOrigin !== "") {
      let parsedOrigin: URL;
      try { parsedOrigin = new URL(rawOrigin); } catch { throw new RangeError("origin must be an absolute HTTP(S) URL, or empty for a same-origin relative request"); }
      if ((parsedOrigin.protocol !== "http:" && parsedOrigin.protocol !== "https:") || parsedOrigin.username || parsedOrigin.password) {
        throw new RangeError("origin must be an absolute HTTP(S) URL without credentials");
      }
    }
    this.origin = rawOrigin.replace(/\/$/, "");
    this.requestFetch = options.fetch ?? fetch;
    const timeout = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    if (!Number.isInteger(timeout) || timeout < 1 || timeout > MAX_TIMEOUT_MS) throw new RangeError("timeoutMs must be between 1 and 120000");
    this.timeoutMs = timeout;
    this.externalSignal = options.signal;
  }

  async getPage(locale: string, slug: string, options?: HydroGoRequestOptions): Promise<Page> {
    return this.getData<Page>(`/api/v1/pages/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`, options);
  }

  async listPages(locale: string, limit = 20, options?: HydroGoRequestOptions): Promise<PageSummary[]> {
    if (!Number.isInteger(limit) || limit < 1 || limit > 100) throw new RangeError("limit must be between 1 and 100");
    return this.getData<PageSummary[]>(`/api/v1/pages?locale=${encodeURIComponent(locale)}&limit=${limit}`, options);
  }

  async listNews(locale: string, limit = 20, options?: HydroGoRequestOptions): Promise<NewsItem[]> {
    if (!Number.isInteger(limit) || limit < 1 || limit > 100) throw new RangeError("limit must be between 1 and 100");
    return this.getData<NewsItem[]>(`/api/v1/news?locale=${encodeURIComponent(locale)}&limit=${limit}`, options);
  }

  // One page of news with pagination metadata (listNews returns only the items).
  async listNewsPage(locale: string, limit = 20, offset = 0, options?: HydroGoRequestOptions): Promise<NewsPage> {
    if (!Number.isInteger(limit) || limit < 1 || limit > 100) throw new RangeError("limit must be between 1 and 100");
    if (!Number.isInteger(offset) || offset < 0) throw new RangeError("offset must be 0 or greater");
    return this.request<NewsPage>(`/api/v1/news?locale=${encodeURIComponent(locale)}&limit=${limit}&offset=${offset}`, {}, options);
  }

  // URL of an uploaded media file (the id comes from fields such as cover_media_id).
  mediaUrl(id: string): string {
    return `${this.origin}/media/${encodeURIComponent(id)}`;
  }

  async getNews(locale: string, slug: string, options?: HydroGoRequestOptions): Promise<NewsItem> {
    return this.getData<NewsItem>(`/api/v1/news/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`, options);
  }

  // Content types defined in content-types/*.yaml (published records only).
  // The API returns {data, metadata}; list returns it whole so callers can page.
  async listContent<T = Record<string, unknown>>(type: string, query: ContentQuery = {}, options?: HydroGoRequestOptions): Promise<ContentList<T>> {
    const params = new URLSearchParams();
    if (query.limit !== undefined) {
      if (!Number.isInteger(query.limit) || query.limit < 1 || query.limit > 100) throw new RangeError("limit must be between 1 and 100");
      params.set("limit", String(query.limit));
    }
    if (query.offset !== undefined) params.set("offset", String(query.offset));
    if (query.after) params.set("after", query.after);
    if (query.expand?.length) params.set("expand", query.expand.join(","));
    if (query.sort) params.set("sort", query.sort);
    if (query.search) params.set("search", query.search);
    if (query.locale) params.set("locale", query.locale);
    for (const [field, value] of Object.entries(query.filter ?? {})) {
      if (typeof value === "string") params.set(`filter[${field}]`, value);
      else for (const [op, operand] of Object.entries(value)) if (operand !== undefined) params.set(`filter[${field}][${op}]`, operand);
    }
    const suffix = params.size > 0 ? `?${params}` : "";
    return this.request<ContentList<T>>(`/api/v1/content/${encodeURIComponent(type)}${suffix}`, {}, options);
  }

  async getContent<T = Record<string, unknown>>(type: string, id: string, locale?: string, options?: HydroGoRequestOptions): Promise<T> {
    const suffix = locale ? `?locale=${encodeURIComponent(locale)}` : "";
    return this.getData<T>(`/api/v1/content/${encodeURIComponent(type)}/${encodeURIComponent(id)}${suffix}`, options);
  }

  // Member methods use only the explicit bearer passed by the host application.
  // They never opt into browser cookies, so a consumer client cannot silently
  // cross the browser-session/staff boundary.
  async memberLogin(input: MemberLoginInput, options?: HydroGoRequestOptions): Promise<MemberTokenPair> {
    return this.request<MemberTokenPair>("/api/v1/mobile/member/auth/login", { method: "POST", credentials: "omit", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }, options);
  }

  async memberRegister(input: MemberLoginInput, options?: HydroGoRequestOptions): Promise<MemberTokenPair> {
    return this.request<MemberTokenPair>("/api/v1/mobile/member/auth/register", { method: "POST", credentials: "omit", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }, options);
  }

  async memberRefresh(refreshToken: string, options?: HydroGoRequestOptions): Promise<MemberTokenPair> {
    if (!refreshToken.trim()) throw new RangeError("refreshToken is required");
    return this.request<MemberTokenPair>("/api/v1/mobile/member/auth/refresh", { method: "POST", credentials: "omit", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ refresh_token: refreshToken }) }, options);
  }

  async memberLogout(accessToken: string, allDevices = false, options?: HydroGoRequestOptions): Promise<void> {
    const path = allDevices ? "/api/v1/mobile/member/auth/revoke-all" : "/api/v1/mobile/member/auth/revoke";
    await this.request<void>(path, { method: "POST", credentials: "omit", headers: this.memberHeaders(accessToken) }, options);
  }

  async memberDevices(accessToken: string, options?: HydroGoRequestOptions): Promise<MemberDevice[]> {
    const response = await this.request<{ devices: MemberDevice[] }>("/api/v1/mobile/member/auth/devices", { credentials: "omit", headers: this.memberHeaders(accessToken) }, options);
    return response.devices;
  }

  async memberMe(accessToken: string, options?: HydroGoRequestOptions): Promise<MemberProfile> {
    return this.request<MemberProfile>("/api/v1/mobile/member/me", { credentials: "omit", headers: this.memberHeaders(accessToken) }, options);
  }

  // Public content endpoints wrap their payload as {data, metadata}.
  private async getData<T>(path: string, options?: HydroGoRequestOptions): Promise<T> {
    const response = await this.request<{ data: T }>(path, {}, options);
    return response.data;
  }

  private memberHeaders(accessToken: string): Record<string, string> {
    if (!accessToken.trim()) throw new RangeError("accessToken is required");
    return { Authorization: `Bearer ${accessToken}` };
  }

  private async request<T>(path: string, init: RequestInit = {}, options?: HydroGoRequestOptions): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    const externalSignal = options?.signal ?? init.signal ?? this.externalSignal;
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
