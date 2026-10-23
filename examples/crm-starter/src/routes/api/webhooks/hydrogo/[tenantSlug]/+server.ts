import { env } from '$env/dynamic/private';
import { json } from '@sveltejs/kit';
import { sendNotification, type NotificationType } from '$lib/server/mailer';
import { verifyWebhookSignature } from '$lib/server/webhook-signature';
import type { RequestHandler } from './$types';

interface WebhookEvent {
  id: string;
  topic: string;
  aggregate_id: string;
  payload_version: number;
}

const NOTIFICATION_TYPE: Record<string, NotificationType> = {
  'content.created': 'created',
  'content.updated': 'updated',
  'content.deleted': 'deleted'
};

/**
 * Receives HydroGo's generic content.* webhook events (fired on every
 * collection record write, no HydroGo core changes needed) and emails a
 * notification. One route per tenant: the webhook payload carries no tenant
 * field, so the secret is looked up by the URL's tenant slug.
 */
export const POST: RequestHandler = async ({ params, request, locals }) => {
  if (params.tenantSlug !== locals.tenant.slug) return json({ error: 'unknown tenant' }, { status: 404 });

  const secret = locals.tenant.webhookSecret ?? env.WEBHOOK_SECRET;
  if (!secret) return json({ error: 'webhook not configured' }, { status: 503 });

  const body = await request.text();
  const ok = verifyWebhookSignature(body, request.headers.get('x-hydro-signature'), request.headers.get('x-hydro-timestamp'), secret);
  if (!ok) return json({ error: 'invalid signature' }, { status: 401 });

  const event = JSON.parse(body) as WebhookEvent;
  const type = NOTIFICATION_TYPE[event.topic];
  if (type) await sendNotification(locals.tenant.slug, type, event.aggregate_id);

  return json({ ok: true });
};
