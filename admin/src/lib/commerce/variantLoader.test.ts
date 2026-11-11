import { describe, expect, it, vi } from 'vitest';
import { api } from '$lib/api/client';
import { chunkIdsByBytes, fetchVariantsForProducts } from './variantLoader';

vi.mock('$lib/api/client', () => ({ api: { get: vi.fn() } }));

const id = (n: number) => `r_${String(n).padStart(30, '0')}`; // 32 chars, like the real ones (r_ + 30 hex)

describe('chunkIdsByBytes', () => {
  it('keeps every joined value under the 256-byte API limit', () => {
    const ids = Array.from({ length: 20 }, (_, i) => id(i));
    const chunks = chunkIdsByBytes(ids);
    expect(chunks.flat()).toEqual(ids);
    for (const c of chunks) expect(c.map(encodeURIComponent).join(',').length).toBeLessThanOrEqual(255);
    expect(chunks.length).toBeGreaterThan(1); // 20 ids cannot fit in one request
    expect(Math.max(...chunks.map((c) => c.length))).toBeLessThanOrEqual(7);
  });

  it('returns no chunks for no ids', () => {
    expect(chunkIdsByBytes([])).toEqual([]);
  });
});

describe('fetchVariantsForProducts', () => {
  it('fetches chunks in parallel and pages a chunk until a short page', async () => {
    const ids = Array.from({ length: 10 }, (_, i) => id(i)); // 2 chunks
    vi.mocked(api.get).mockImplementation((url: string) => {
      const offset = Number(/offset=(\d+)/.exec(url)![1]);
      // first chunk: a full page then a short one; second chunk: one short page
      if (url.includes(ids[0]) && offset === 0) return Promise.resolve({ data: Array.from({ length: 100 }, (_, i) => ({ id: `a${i}` })) });
      if (url.includes(ids[0])) return Promise.resolve({ data: [{ id: 'a100' }] });
      return Promise.resolve({ data: [{ id: 'b0' }] });
    });

    const rows = await fetchVariantsForProducts<{ id: string }>(ids);

    expect(rows).toHaveLength(102);
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(3);
  });
});
