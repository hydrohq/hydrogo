import { createCipheriv, createDecipheriv, randomBytes, scryptSync } from 'node:crypto';
import { Redis } from 'ioredis';
import type { Session } from '../sessions';
import type { SessionStore } from './types';

const ALGO = 'aes-256-gcm';
const SALT = 'hydrogo-crm-session-v1';

/**
 * Shares sessions across panel instances. Independent of HydroGo's own
 * bffsession Redis store (different process, different wire format) — there
 * is nothing on the Go side to stay byte-compatible with.
 */
export class RedisStore implements SessionStore {
  private readonly client: Redis;
  private readonly key: Buffer;
  private readonly prefix: string;

  constructor(redisUrl: string, secret: string, prefix = 'hydro:sess:') {
    this.client = new Redis(redisUrl);
    this.key = scryptSync(secret, SALT, 32);
    this.prefix = prefix;
  }

  async get(key: string): Promise<Session | null> {
    const raw = await this.client.get(this.prefix + key);
    if (!raw) return null;
    try {
      return this.decrypt(raw);
    } catch {
      return null;
    }
  }

  async set(key: string, session: Session, ttlSeconds: number): Promise<void> {
    await this.client.set(this.prefix + key, this.encrypt(session), 'EX', ttlSeconds);
  }

  async delete(key: string): Promise<void> {
    await this.client.del(this.prefix + key);
  }

  private encrypt(session: Session): string {
    const iv = randomBytes(12);
    const cipher = createCipheriv(ALGO, this.key, iv);
    const ciphertext = Buffer.concat([cipher.update(JSON.stringify(session), 'utf8'), cipher.final()]);
    return Buffer.concat([iv, cipher.getAuthTag(), ciphertext]).toString('base64');
  }

  private decrypt(payload: string): Session {
    const raw = Buffer.from(payload, 'base64');
    const iv = raw.subarray(0, 12);
    const tag = raw.subarray(12, 28);
    const ciphertext = raw.subarray(28);
    const decipher = createDecipheriv(ALGO, this.key, iv);
    decipher.setAuthTag(tag);
    const plaintext = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
    return JSON.parse(plaintext.toString('utf8'));
  }
}
