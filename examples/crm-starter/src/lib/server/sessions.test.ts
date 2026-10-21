import { afterEach, describe, expect, it, vi } from 'vitest';
import { createSession, destroySession, getSession, resetSessions } from './sessions';

afterEach(() => {
  resetSessions();
  vi.useRealTimers();
});

describe('sessions', () => {
  const data = { cookie: 'hydro_session=a', csrf: 'x', email: 'a@b.co' };

  it('issues unguessable ids and returns the stored session', () => {
    const a = createSession(data);
    const b = createSession(data);
    expect(a).not.toEqual(b);
    expect(a.length).toBeGreaterThanOrEqual(40);
    expect(getSession(a)?.email).toBe('a@b.co');
  });

  it('forgets destroyed and expired sessions', () => {
    vi.useFakeTimers();
    const sid = createSession(data);
    const other = createSession(data);
    destroySession(sid);
    expect(getSession(sid)).toBeNull();
    vi.setSystemTime(Date.now() + 9 * 60 * 60 * 1000);
    expect(getSession(other)).toBeNull();
    expect(getSession(undefined)).toBeNull();
  });
});
