import { createHmac } from 'node:crypto';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { verifyWebhookSignature } from './webhook-signature';

function sign(body: string, secret: string, timestamp: string): string {
  return 'sha256=' + createHmac('sha256', secret).update(`${timestamp}.`).update(body).digest('hex');
}

afterEach(() => vi.useRealTimers());

describe('verifyWebhookSignature', () => {
  const secret = 's3cret';
  const body = JSON.stringify({ id: 'e1', topic: 'content.updated', aggregate_id: 'deals/1', payload_version: 1 });

  it('accepts a signature matching internal/webhooks/dispatcher.go\'s scheme', () => {
    const timestamp = String(Math.floor(Date.now() / 1000));
    expect(verifyWebhookSignature(body, sign(body, secret, timestamp), timestamp, secret)).toBe(true);
  });

  it('rejects a wrong secret, a tampered body, or a missing header', () => {
    const timestamp = String(Math.floor(Date.now() / 1000));
    expect(verifyWebhookSignature(body, sign(body, 'other', timestamp), timestamp, secret)).toBe(false);
    expect(verifyWebhookSignature(body + 'x', sign(body, secret, timestamp), timestamp, secret)).toBe(false);
    expect(verifyWebhookSignature(body, null, timestamp, secret)).toBe(false);
    expect(verifyWebhookSignature(body, sign(body, secret, timestamp), null, secret)).toBe(false);
  });

  it('rejects a stale timestamp', () => {
    const timestamp = String(Math.floor(Date.now() / 1000) - 10 * 60);
    expect(verifyWebhookSignature(body, sign(body, secret, timestamp), timestamp, secret)).toBe(false);
  });
});
