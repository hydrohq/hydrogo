import { describe, it, expect, vi, beforeEach } from 'vitest';
import { loadHookServices } from './services';

describe('loadHookServices', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('returns the services of the envelope', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ data: { services: [{ id: 'ledger', host: 'x', schedules: [], used_by: [], calls: 0, failures: 0 }] } }),
    });
    expect((await loadHookServices())[0].id).toBe('ledger');
  });

  it('is empty when the endpoint does not exist (no hooks configured)', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      headers: new Headers(),
      json: async () => ({ code: 'not_found', message: 'not found' }),
    });
    expect(await loadHookServices()).toEqual([]);
  });
});
