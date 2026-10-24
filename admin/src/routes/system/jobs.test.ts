import { describe, it, expect, vi, beforeEach } from 'vitest';
import { loadJobs, loadDeadLetterJobs, requeueJob, deleteDeadLetterJob } from './jobs';

describe('jobs admin client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('returns the jobs of the envelope', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ data: { jobs: [{ id: 'job-1', type: 'news.publish', payload: '{}', available_at: '', created_at: '', attempts: 0, dead_lettered: false }] } }),
    });
    expect((await loadJobs())[0].id).toBe('job-1');
  });

  it('returns dead-letter jobs of the envelope', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ data: { jobs: [{ id: 'job-2', type: 'news.publish', payload: '{}', available_at: '', created_at: '', attempts: 3, dead_lettered: true, last_error: 'boom' }] } }),
    });
    expect((await loadDeadLetterJobs())[0].last_error).toBe('boom');
  });

  it('requeues a job by posting to its requeue endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ data: { id: 'job-2' } }),
    });
    globalThis.fetch = fetchMock;
    await requeueJob('job-2');
    const [url, init] = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    expect(String(url)).toContain('/api/v1/admin/jobs/dead-letters/job-2/requeue');
    expect(init?.method).toBe('POST');
  });

  it('deletes a dead-letter job', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ data: { id: 'job-2' } }),
    });
    globalThis.fetch = fetchMock;
    await deleteDeadLetterJob('job-2');
    const [url, init] = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    expect(String(url)).toContain('/api/v1/admin/jobs/dead-letters/job-2');
    expect(init?.method).toBe('DELETE');
  });
});
