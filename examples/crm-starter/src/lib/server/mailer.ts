import { env } from '$env/dynamic/private';
import nodemailer from 'nodemailer';

let transport: ReturnType<typeof nodemailer.createTransport> | null = null;

function smtp() {
  if (transport) return transport;
  if (!env.SMTP_HOST) throw new Error('SMTP_HOST is required to send notifications');
  transport = nodemailer.createTransport({
    host: env.SMTP_HOST,
    port: Number(env.SMTP_PORT ?? 587),
    auth: env.SMTP_USER ? { user: env.SMTP_USER, pass: env.SMTP_PASS } : undefined
  });
  return transport;
}

export type NotificationType = 'created' | 'updated' | 'deleted';

/**
 * v1 can only say a record changed, not what changed: HydroGo's content.*
 * webhook events carry no field-level diff, so "deal stage changed" can't be
 * told apart from any other deal edit yet.
 */
export async function sendNotification(tenantSlug: string, type: NotificationType, aggregateId: string): Promise<void> {
  const to = env.NOTIFY_TO;
  if (!to) return;
  const [collection, id] = aggregateId.split('/');
  await smtp().sendMail({
    from: env.SMTP_FROM ?? 'crm@localhost',
    to,
    subject: `[${tenantSlug}] ${collection} ${type}`,
    text: `${collection} record ${id} was ${type} in HydroGo.`
  });
}
