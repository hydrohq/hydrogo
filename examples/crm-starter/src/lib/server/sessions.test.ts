import { afterEach, describe, expect, it, vi } from 'vitest';
import { createSession, destroySession, getSession, resetSessions } from './sessions';

afterEach(() => {
  resetSessions();
  vi.useRealTimers();
});

describe('sessions', () => {
  const data = { cookie: 'hydro_session=a', csrf: 'x', email: 'a@b.co', tenantId: 'default', hydroOrigin: 'http://127.0.0.1:8080' };

  it('issues unguessable ids and returns the stored session', async () => {
    const a = await createSession(data);
    const b = await createSession(data);
    expect(a).not.toEqual(b);
    expect(a.length).toBeGreaterThanOrEqual(40);
    expect((await getSession(a))?.email).toBe('a@b.co');
  });

  it('forgets destroyed and expired sessions', async () => {
    vi.useFakeTimers();
    const sid = await createSession(data);
    const other = await createSession(data);
    destroySession(sid);
    expect(await getSession(sid)).toBeNull();
    vi.setSystemTime(Date.now() + 9 * 60 * 60 * 1000);
    expect(await getSession(other)).toBeNull();
    expect(await getSession(undefined)).toBeNull();
  });
});
