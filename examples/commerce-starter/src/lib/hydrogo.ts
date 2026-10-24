// A trimmed copy of packages/client's generic content-type client: only
// what a storefront's catalog pages need (listContent/getContent/mediaUrl).
// Cart, checkout and payment go through src/lib/server/commerce.ts instead,
// since those need cookie forwarding this client does not do.

export type HydroAPIErrorBody = { code?: string; message?: string; error?: string; request_id?: string };

export class HydroGoError extends Error {
  readonly code: string;
  readonly requestId: string;
  constructor(readonly status: number, readonly body?: HydroAPIErrorBody, requestId = '') {
    super(body?.message ?? body?.error ?? `HydroGo API returned ${status}`);
    this.name = 'HydroGoError';
    this.code = body?.code ?? 'api_error';
    this.requestId = body?.request_id ?? requestId;
  }
}

export type FilterOp = 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'null';
export type FilterValue = string | Partial<Record<FilterOp, string>>;
export type ContentQuery = { limit?: number; offset?: number; after?: string; sort?: string; search?: string; expand?: string[]; filter?: Record<string, FilterValue> };
export type ContentList<T> = { data: T[]; metadata: { limit: number; has_more: boolean; next_cursor?: string } };
export type HydroGoClientOptions = { origin: string; fetch?: typeof fetch; timeoutMs?: number };

const DEFAULT_TIMEOUT_MS = 15_000;
const MAX_ERROR_BODY_CODEPOINTS = 8_192;
const MAX_RESPONSE_BYTES = 2 * 1024 * 1024;

async function readBoundedResponse(response: Response): Promise<string> {
  const text = await response.text();
  if (new TextEncoder().encode(text).byteLength > MAX_RESPONSE_BYTES) {
    throw new HydroGoError(response.status, { code: 'response_too_large', message: 'HydroGo API response is too large' }, response.headers.get('x-request-id') ?? '');
  }
  return text;
}

export class HydroGoClient {
  private readonly origin: string;
  private readonly requestFetch: typeof fetch;
  private readonly timeoutMs: number;

  constructor(options: HydroGoClientOptions) {
    const rawOrigin = options.origin.trim();
    let parsedOrigin: URL;
    try {
      parsedOrigin = new URL(rawOrigin);
    } catch {
      throw new RangeError('origin must be an absolute HTTP(S) URL');
    }
    if (parsedOrigin.protocol !== 'http:' && parsedOrigin.protocol !== 'https:') throw new RangeError('origin must be an absolute HTTP(S) URL');
    this.origin = rawOrigin.replace(/\/$/, '');
    this.requestFetch = options.fetch ?? fetch;
    this.timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  }

  mediaUrl(id: string): string {
    return `${this.origin}/media/${encodeURIComponent(id)}`;
  }

  async listContent<T = Record<string, unknown>>(type: string, query: ContentQuery = {}): Promise<ContentList<T>> {
    const params = new URLSearchParams();
    if (query.limit !== undefined) params.set('limit', String(query.limit));
    if (query.offset !== undefined) params.set('offset', String(query.offset));
    if (query.after) params.set('after', query.after);
    if (query.expand?.length) params.set('expand', query.expand.join(','));
    if (query.sort) params.set('sort', query.sort);
    if (query.search) params.set('search', query.search);
    for (const [field, value] of Object.entries(query.filter ?? {})) {
      if (typeof value === 'string') params.set(`filter[${field}]`, value);
      else for (const [op, operand] of Object.entries(value)) if (operand !== undefined) params.set(`filter[${field}][${op}]`, operand);
    }
    const suffix = params.size > 0 ? `?${params}` : '';
    return this.request<ContentList<T>>(`/api/v1/content/${encodeURIComponent(type)}${suffix}`);
  }

  async getContent<T = Record<string, unknown>>(type: string, id: string): Promise<T> {
    const response = await this.request<{ data: T }>(`/api/v1/content/${encodeURIComponent(type)}/${encodeURIComponent(id)}`);
    return response.data;
  }

  private async request<T>(path: string): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    try {
      const response = await this.requestFetch(`${this.origin}${path}`, { signal: controller.signal });
      if (!response.ok) {
        const raw = [...(await readBoundedResponse(response))].slice(0, MAX_ERROR_BODY_CODEPOINTS).join('');
        let body: HydroAPIErrorBody | undefined;
        try {
          body = JSON.parse(raw) as HydroAPIErrorBody;
        } catch {
          /* keep body absent */
        }
        throw new HydroGoError(response.status, body, response.headers.get('x-request-id') ?? '');
      }
      return JSON.parse(await readBoundedResponse(response)) as T;
    } finally {
      clearTimeout(timer);
    }
  }
}

export function createHydroGoClient(options: HydroGoClientOptions): HydroGoClient {
  return new HydroGoClient(options);
}
