import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ApiClientError } from '$lib/api/client';
import { importCsvFile, importErrorMessage } from './csvImport';

describe('csvImport', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('sends the file as text/csv and returns the created count', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ data: { rows: 2, created: 2, dry_run: false } }),
    });
    const created = await importCsvFile('/api/v1/admin/content/faq', new File(['question\nA\nB\n'], 'faq.csv'));
    expect(created).toBe(2);
    const calls = (globalThis.fetch as any).mock.calls;
    const [url, options] = calls[calls.length - 1];
    expect(url).toContain('/api/v1/admin/content/faq/_import');
    expect(options.method).toBe('POST');
    expect(options.headers.get('Content-Type')).toBe('text/csv');
    expect(options.body).toBe('question\nA\nB\n');
  });

  it('shows the message, the first three field errors and the request id', () => {
    const err = new ApiClientError(
      { code: 'invalid', message: 'Invalid file', request_id: 'req-9', fields: { 'rows[0].a': 'x', 'rows[1].b': 'y', 'rows[2].c': 'z', 'rows[3].d': 'w' } },
      422,
    );
    expect(importErrorMessage(err, 'Request id', 'failed')).toBe('Invalid file — rows[0].a: x; rows[1].b: y; rows[2].c: z (Request id: req-9)');
  });

  it('falls back for errors that are not API errors', () => {
    expect(importErrorMessage(new Error('boom'), 'Request id', 'failed')).toBe('failed');
  });
});
