import { randomBytes } from 'node:crypto';
import { env } from '$env/dynamic/private';
import { MemoryStore } from './session-store/memory-store';
import { RedisStore } from './session-store/redis-store';
import type { SessionStore } from './session-store/types';

/**
 * The browser only holds an opaque panel session id. HydroGo's own session
 * cookie and CSRF token stay here, on the server, so a script injected into
 * the panel can never read them.
 *
 * Backed by an in-memory store by default (fine for one panel process;
 * restart = everyone signs in again). Set REDIS_URL (+ SESSION_SECRET) to
 * share sessions across instances.
 */
export interface Session {
  /** Cookie header value for HydroGo: "hydro_session=…; hydro_csrf=…". */
  cookie: string;
  csrf: string;
  email: string;
  /** Which tenant this session was created for; checked against the request's resolved tenant on every read. */
  tenantId: string;
  /** The HydroGo origin this session's cookie/csrf pair is valid against. */
  hydroOrigin: string;
}

const TTL_SECONDS = 8 * 60 * 60;

function buildStore(): SessionStore {
  if (!env.REDIS_URL) return new MemoryStore();
  if (!env.SESSION_SECRET) throw new Error('SESSION_SECRET is required when REDIS_URL is set');
  return new RedisStore(env.REDIS_URL, env.SESSION_SECRET);
}

const store = buildStore();

export async function createSession(data: Session): Promise<string> {
  const sid = randomBytes(32).toString('base64url');
  await store.set(sid, data, TTL_SECONDS);
  return sid;
}

export async function getSession(sid: string | undefined): Promise<Session | null> {
  if (!sid) return null;
  return store.get(sid);
}

/** Fire-and-forget: callers don't need to await cleanup, the TTL expires stale entries anyway. */
export function destroySession(sid: string | undefined): void {
  if (sid) void store.delete(sid).catch(() => undefined);
}

export const sessionTtlSeconds = TTL_SECONDS;

/** For tests. */
export function resetSessions(): void {
  if (store instanceof MemoryStore) store.reset();
}
