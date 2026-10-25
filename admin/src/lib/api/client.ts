import type { ApiError } from './types';

class ApiClientError extends Error {
  code: string;
  requestId: string;
  fields?: Record<string, string>;
  status: number;

  constructor(error: ApiError, status: number) {
    super(error.message || 'API request failed');
    this.name = 'ApiClientError';
    this.code = error.code || 'UNKNOWN_ERROR';
    this.requestId = error.request_id || '';
    this.fields = error.fields;
    this.status = status;
  }
}

export { ApiClientError };

let csrfToken: string | null = null;

export function setCsrfToken(token: string | null) {
  csrfToken = token;
}

export function getCsrfToken(): string | null {
  return csrfToken;
}

export async function fetchCsrfToken(): Promise<string> {
  try {
    const res = await fetch('/csrf-token', {
      method: 'GET',
      credentials: 'same-origin',
    });
    if (res.ok) {
      const data = await res.json();
      const token = data.token || data.csrf_token || (data.data && data.data.token);
      if (token) {
        csrfToken = token;
        return token;
      }
    }
  } catch (e) {
    console.warn('Failed to fetch CSRF token:', e);
  }
  return '';
}

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
  /** A 401 here is an answer to show (a bad link), not a lost session to redirect from. */
  noAuthRedirect?: boolean;
}

export async function apiRequest<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const base = typeof window !== 'undefined' ? window.location.origin : 'http://localhost';
  const url = new URL(endpoint.startsWith('/') ? endpoint : `/${endpoint}`, base);

  if (options.params) {
    Object.entries(options.params).forEach(([key, value]) => {
      if (value !== undefined) {
        url.searchParams.append(key, String(value));
      }
    });
  }

  const headers = new Headers(options.headers || {});

  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }
  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json');
  }
  if (!headers.has('X-Requested-With')) {
    headers.set('X-Requested-With', 'XMLHttpRequest');
  }

  // Double-submit CSRF protection for mutating methods
  const method = (options.method || 'GET').toUpperCase();
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
    if (!csrfToken) {
      await fetchCsrfToken();
    }
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken);
    }
  }

  const response = await fetch(url.toString(), {
    ...options,
    headers,
    credentials: 'include',
  });

  if (response.status === 401 && !options.noAuthRedirect && typeof window !== 'undefined') {
    // If we're unauthorized and not already on a page that is meant to be reached
    // signed out (sign-in, set a password from an e-mailed link), redirect.
    if (!isSignedOutPage(window.location.pathname, window.location.hash)) {
      const currentPath = window.location.pathname + window.location.search + window.location.hash;
      const redirectUrl = `/admin/login?redirect=${encodeURIComponent(currentPath)}`;
      window.location.href = redirectUrl;
    }
  }

  // Extract CSRF token from header if server returned a new one
  const newCsrf = response.headers.get('X-CSRF-Token');
  if (newCsrf) {
    csrfToken = newCsrf;
  }

  if (response.status === 204) {
    return {} as T;
  }

  let body: any;
  try {
    body = await response.json();
  } catch (err) {
    if (!response.ok) {
      throw new ApiClientError({
        code: `HTTP_${response.status}`,
        message: response.statusText || 'Unknown network error',
        request_id: response.headers.get('X-Request-Id') || '',
      }, response.status);
    }
    return {} as T;
  }

  if (!response.ok) {
    const error: ApiError = {
      code: body.code || `HTTP_${response.status}`,
      message: body.message || 'An error occurred',
      request_id: body.request_id || response.headers.get('X-Request-Id') || '',
      fields: body.fields,
    };
    throw new ApiClientError(error, response.status);
  }

  // Envelope unwrapping: if payload has { data: ... }
  if (body && typeof body === 'object' && 'data' in body) {
    return body as T;
  }

  return body as T;
}

const SIGNED_OUT_PAGES = ['/login', '/reset-password', '/accept-invite'];

/** Pages a person reaches without a session; a 401 there is expected, not a reason to redirect. */
export function isSignedOutPage(pathname: string, hash = ''): boolean {
  return SIGNED_OUT_PAGES.some((page) => pathname.includes(page) || hash.includes(page));
}

/**
 * FormData and strings go out as they are (a string is already encoded, for
 * example a URLSearchParams for the endpoints that read form fields); anything
 * else is sent as JSON. Quoting a string a second time made those endpoints
 * read an empty form.
 */
const encodeBody = (body: unknown): BodyInit | undefined =>
  body instanceof FormData || typeof body === 'string' ? body : JSON.stringify(body);

export const api = {
  get: <T>(endpoint: string, options?: RequestOptions) =>
    apiRequest<T>(endpoint, { ...options, method: 'GET' }),

  post: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'POST',
      body: encodeBody(body),
    }),

  put: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: encodeBody(body),
    }),

  patch: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'PATCH',
      body: encodeBody(body),
    }),

  delete: <T>(endpoint: string, options?: RequestOptions) =>
    apiRequest<T>(endpoint, { ...options, method: 'DELETE' }),

  upload: <T>(endpoint: string, formData: FormData, options?: RequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'POST',
      body: formData,
    }),
};
