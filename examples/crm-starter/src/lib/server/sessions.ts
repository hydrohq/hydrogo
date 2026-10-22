import { randomBytes } from 'node:crypto';

/**
 * The browser only holds an opaque panel session id. HydroGo's own session
 * cookie and CSRF token stay here, on the server, so a script injected into
 * the panel can never read them.
 *
 * The store is in memory: fine for one panel process; restart = everyone signs
 * in again. Run several instances only behind sticky sessions, or replace this
 * module with a shared store (Redis, a database table).
 */
export interface Session {
  /** Cookie header value for HydroGo: "hydro_session=…; hydro_csrf=…". */
  cookie: string;
  csrf: string;
  email: string;
  expires: number;
}

const TTL_MS = 8 * 60 * 60 * 1000;
const MAX_SESSIONS = 5000;
const store = new Map<string, Session>();

export function createSession(data: Omit<Session, 'expires'>): string {
  prune();
  if (store.size >= MAX_SESSIONS) {
    const oldest = store.keys().next().value;
    if (oldest) store.delete(oldest);
  }
  const sid = randomBytes(32).toString('base64url');
  store.set(sid, { ...data, expires: Date.now() + TTL_MS });
  return sid;
}

export function getSession(sid: string | undefined): Session | null {
  if (!sid) return null;
  const session = store.get(sid);
  if (!session) return null;
  if (session.expires < Date.now()) {
    store.delete(sid);
    return null;
  }
  return session;
}

export function destroySession(sid: string | undefined) {
  if (sid) store.delete(sid);
}

export const sessionTtlSeconds = TTL_MS / 1000;

function prune() {
  const now = Date.now();
  for (const [sid, session] of store) if (session.expires < now) store.delete(sid);
}

/** For tests. */
export function resetSessions() {
  store.clear();
}
