import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api } from './client';

describe('ApiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('unwraps data envelopes on successful 200 response', async () => {
    const payload = { id: 'item-1', name: 'Test Article' };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ data: payload, meta: { total: 1 } }),
    });

    const res = await api.get('/api/v1/content/faq');
    expect(res).toEqual({ data: payload, meta: { total: 1 } });
    expect(globalThis.fetch).toHaveBeenCalled();

    const [calledUrl, options] = (globalThis.fetch as any).mock.calls[0];
    expect(calledUrl).toContain('/api/v1/content/faq');
    expect(options.headers.get('Accept')).toBe('application/json');
    expect(options.headers.get('X-Requested-With')).toBe('XMLHttpRequest');
  });

  it('throws ApiClientError with code and message on error envelopes', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({
        code: 'VALIDATION_FAILED',
        message: 'Title is required',
        request_id: 'req-12345',
        fields: { title: 'required' },
      }),
    });

    await expect(api.post('/api/v1/content/faq', {})).rejects.toThrow('Title is required');
  });

  it('handles query parameters accurately', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ data: [] }),
    });

    await api.get('/api/v1/admin/audit', { params: { actor: 'admin@acme.com', limit: 20 } });
    const [calledUrl] = (globalThis.fetch as any).mock.calls[0];
    expect(calledUrl).toContain('/api/v1/admin/audit?actor=admin%40acme.com&limit=20');
  });
});
