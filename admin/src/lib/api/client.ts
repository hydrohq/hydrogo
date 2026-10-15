import type { ApiResponse, ApiError } from './types';

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
}

export async function apiRequest<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const url = new URL(endpoint.startsWith('/') ? endpoint : `/${endpoint}`, window.location.origin);

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

  if (response.status === 401) {
    // If we're unauthorized and not already on the login page, redirect
    if (!window.location.pathname.includes('/login') && !window.location.hash.includes('/login')) {
      const currentPath = window.location.pathname + window.location.search + window.location.hash;
      const redirectUrl = `/admin/#/login?redirect=${encodeURIComponent(currentPath)}`;
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

export const api = {
  get: <T>(endpoint: string, options?: RequestOptions) =>
    apiRequest<T>(endpoint, { ...options, method: 'GET' }),

  post: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body instanceof FormData ? body : JSON.stringify(body),
    }),

  put: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body instanceof FormData ? body : JSON.stringify(body),
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
