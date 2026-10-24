import { createHmac, timingSafeEqual } from 'node:crypto';

const MAX_SKEW_SECONDS = 5 * 60;

/**
 * Matches internal/webhooks/dispatcher.go's signatureAt: HMAC-SHA256 over
 * "<unix-seconds>." + body, hex-encoded, prefixed "sha256=".
 */
export function verifyWebhookSignature(body: string, signatureHeader: string | null, timestampHeader: string | null, secret: string): boolean {
  if (!signatureHeader || !timestampHeader || !secret) return false;
  const timestamp = Number(timestampHeader);
  if (!Number.isFinite(timestamp) || Math.abs(Date.now() / 1000 - timestamp) > MAX_SKEW_SECONDS) return false;

  const expected = 'sha256=' + createHmac('sha256', secret).update(`${timestampHeader}.`).update(body).digest('hex');
  const a = Buffer.from(expected);
  const b = Buffer.from(signatureHeader);
  return a.length === b.length && timingSafeEqual(a, b);
}
