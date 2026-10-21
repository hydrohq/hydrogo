import { describe, it, expect, vi, beforeEach } from 'vitest';
import { schema } from './schema.svelte';

function envelope(data: unknown) {
  return { ok: true, status: 200, headers: new Headers(), json: async () => ({ data }) };
}

describe('schema.loadSchema', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    schema.schema = null;
    schema.error = null;
  });

  it('shares one request between concurrent callers', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope({ modules: { pages: true }, resources: [] }));
    globalThis.fetch = fetchMock;
    await Promise.all([schema.loadSchema(), schema.loadSchema(true)]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(schema.modules.pages).toBe(true);
    expect(schema.isLoading).toBe(false);
  });

  it('skips the request when loaded, unless forced', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope({ modules: {}, resources: [] }));
    globalThis.fetch = fetchMock;
    await schema.loadSchema();
    await schema.loadSchema();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    await schema.loadSchema(true);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
