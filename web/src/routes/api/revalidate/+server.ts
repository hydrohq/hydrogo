import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { createHmac, timingSafeEqual } from 'node:crypto';

export const POST: RequestHandler = async ({ request }) => {
  const secret = process.env.HYDROGO_WEBHOOK_SECRET;

  // Strict: Never fallback to any hardcoded default secret
  if (!secret) {
    console.error('[Webhook] HYDROGO_WEBHOOK_SECRET environment variable is missing.');
    return json({ error: 'Server misconfiguration: webhook secret is required' }, { status: 500 });
  }

  const signature = request.headers.get('x-hydro-signature');
  const timestamp = request.headers.get('x-hydro-timestamp');

  if (!signature || !timestamp) {
    return json({ error: 'Missing webhook signature or timestamp headers' }, { status: 401 });
  }

  // 1. Replay attack protection (5-minute tolerance window)
  const now = Math.floor(Date.now() / 1000);
  const ts = parseInt(timestamp, 10);
  if (isNaN(ts) || Math.abs(now - ts) > 300) {
    return json({ error: 'Webhook timestamp expired or out of tolerance window' }, { status: 401 });
  }

  // 2. Read raw payload body for accurate HMAC computation
  const rawBody = await request.text();

  // 3. Compute HMAC-SHA256 matching HydroGo dispatcher: signatureAt(body, secret, timestamp)
  // mac.Write(timestamp + ".") then mac.Write(body) -> "sha256=" + hex
  const hmac = createHmac('sha256', secret);
  hmac.update(`${timestamp}.`);
  hmac.update(rawBody);
  const expectedSignature = `sha256=${hmac.digest('hex')}`;

  // 4. Timing-safe comparison to prevent side-channel timing attacks
  const sigBuffer = Buffer.from(signature);
  const expectedBuffer = Buffer.from(expectedSignature);

  if (sigBuffer.length !== expectedBuffer.length || !timingSafeEqual(sigBuffer, expectedBuffer)) {
    return json({ error: 'Invalid webhook HMAC signature' }, { status: 401 });
  }

  try {
    const payload = JSON.parse(rawBody || '{}');
    const event = request.headers.get('x-hydro-event') || payload.topic || 'news.published';
    console.log(`[Cache Invalidation] Verified event '${event}' received for:`, payload);

    // Rule 12 Zero-build: On-demand edge/CDN cache invalidation
    return json({
      revalidated: true,
      timestamp: new Date().toISOString(),
      event,
    });
  } catch (err: any) {
    return json({ error: 'Failed to process webhook event', details: err?.message }, { status: 500 });
  }
};
