import type { Session } from '../sessions';
import type { SessionStore } from './types';

const MAX_SESSIONS = 5000;

interface Entry {
  session: Session;
  expiresAt: number;
}

/** Fine for one process; restart = everyone signs in again. Default when REDIS_URL isn't set. */
export class MemoryStore implements SessionStore {
  private entries = new Map<string, Entry>();

  async get(key: string): Promise<Session | null> {
    this.prune();
    const entry = this.entries.get(key);
    if (!entry) return null;
    if (entry.expiresAt < Date.now()) {
      this.entries.delete(key);
      return null;
    }
    return entry.session;
  }

  async set(key: string, session: Session, ttlSeconds: number): Promise<void> {
    this.prune();
    if (this.entries.size >= MAX_SESSIONS) {
      const oldest = this.entries.keys().next().value;
      if (oldest) this.entries.delete(oldest);
    }
    this.entries.set(key, { session, expiresAt: Date.now() + ttlSeconds * 1000 });
  }

  async delete(key: string): Promise<void> {
    this.entries.delete(key);
  }

  /** For tests. */
  reset(): void {
    this.entries.clear();
  }

  private prune(): void {
    const now = Date.now();
    for (const [key, entry] of this.entries) if (entry.expiresAt < now) this.entries.delete(key);
  }
}
