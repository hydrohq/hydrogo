import type { Session } from '../sessions';

/** Pluggable session backend: in-memory for one process, Redis to share across instances. */
export interface SessionStore {
  get(key: string): Promise<Session | null>;
  set(key: string, session: Session, ttlSeconds: number): Promise<void>;
  delete(key: string): Promise<void>;
}
