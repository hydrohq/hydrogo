import { describe, it, expect, vi, beforeEach } from 'vitest';

const get = vi.fn();
const post = vi.fn();
vi.mock('$lib/api/client', () => ({ api: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) } }));

import { hiddenStatusField, setPublication, usesDrafts } from './publication';

describe('publication', () => {
  beforeEach(() => { get.mockReset(); post.mockReset(); });

  it('usesDrafts is true only for drafts types without a custom workflow', () => {
    expect(usesDrafts({ drafts: true })).toBe(true);
    expect(usesDrafts({ drafts: true, workflow: { enabled: true } })).toBe(false);
    expect(usesDrafts({})).toBe(false);
    expect(usesDrafts(null)).toBe(false);
  });

  it('recognises the server shape (workflow over a status field, no drafts flag) and hides the raw status field', () => {
    const served = { workflow: { state_field: 'status', states: ['draft', 'published'] } };
    expect(usesDrafts(served)).toBe(true);
    expect(hiddenStatusField(served)).toBe('status');
    expect(hiddenStatusField({})).toBe('');
  });

  it('publishes through the workflow endpoint with the record version', async () => {
    await setPublication('/api/v1/admin/content/reviews', 'r1', 'published', { status: 'draft', version: 3 });
    expect(post).toHaveBeenCalledWith('/api/v1/admin/content/reviews/r1/workflow', { target: 'published', version: 3 });
  });

  it('reads the record first when status/version are unknown, and treats a missing status as draft', async () => {
    get.mockResolvedValue({ data: { status: undefined, version: 2 } });
    await setPublication('/e', 'r1', 'published');
    expect(post).toHaveBeenCalledWith('/e/r1/workflow', { target: 'published', version: 2 });
  });

  it('does nothing when already in the target state', async () => {
    await setPublication('/e', 'r1', 'published', { status: 'published', version: 4 });
    expect(post).not.toHaveBeenCalled();
  });
});
