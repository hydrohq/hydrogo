import { describe, expect, it, vi } from 'vitest';
import { handle } from './hooks.server';

// GHSA-pfhg-q3qv-9q6r (CVE-2026-114060): a `//`-prefixed path must never
// reach route resolution, since @sveltejs/kit < 3.0.0 can turn that into an
// open redirect through use:enhance.
describe('handle', () => {
  it('rejects a //-prefixed path before resolving the route', async () => {
    const resolve = vi.fn();
    const event = { url: new URL('https://example.test//evil.com/path') } as Parameters<typeof handle>[0]['event'];
    const response = await handle({ event, resolve });
    expect(response.status).toBe(404);
    expect(resolve).not.toHaveBeenCalled();
  });
});
