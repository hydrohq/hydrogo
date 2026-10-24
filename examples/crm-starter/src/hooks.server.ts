import { env } from '$env/dynamic/private';
import { redirect, type Handle, type HandleServerError } from '@sveltejs/kit';
import { pickLang } from '$lib/i18n';
import { getSession } from '$lib/server/sessions';
import { resolveTenant } from '$lib/server/tenant';

export const SESSION_COOKIE = 'crm_sid';
const PUBLIC_PATHS = new Set(['/login', '/health']);

export const handle: Handle = async ({ event, resolve }) => {
  // GHSA-pfhg-q3qv-9q6r (CVE-2026-114060): @sveltejs/kit < 3.0.0 can turn a
  // `//`-prefixed action path into an open redirect through use:enhance.
  // This is the SvelteKit security team's own documented workaround --
  // reject it outright before route resolution.
  if (event.url.pathname.startsWith('//')) {
    return new Response('Not found', { status: 404 });
  }

  // Infra health checks hit this from hosts that aren't a configured tenant; mirrors
  // internal/tenant.HostResolver's own /healthz bypass.
  if (event.url.pathname === '/health') return resolve(event);

  const tenant = resolveTenant(event.request.headers.get('host'));
  if (!tenant) return new Response('Unknown host', { status: 404 });
  event.locals.tenant = tenant;

  const sid = event.cookies.get(SESSION_COOKIE) ?? null;
  event.locals.sid = sid;
  const session = await getSession(sid ?? undefined);
  // A session is only valid for the tenant it was created under, even if the sid itself is still live.
  event.locals.session = session && session.tenantId === tenant.id ? session : null;
  event.locals.lang = pickLang(event.cookies.get('crm_lang'), pickLang(env.CRM_LANG));

  // The webhook receiver authenticates itself via HMAC signature, not a panel session.
  const isWebhook = event.url.pathname.startsWith('/api/webhooks/');
  if (!event.locals.session && !isWebhook && !PUBLIC_PATHS.has(event.url.pathname)) {
    if (sid) event.cookies.delete(SESSION_COOKIE, { path: '/' });
    redirect(303, '/login');
  }

  const response = await resolve(event, { transformPageChunk: ({ html }) => html.replace('%lang%', event.locals.lang) });
  response.headers.set('x-content-type-options', 'nosniff');
  response.headers.set('referrer-policy', 'same-origin');
  response.headers.set('x-frame-options', 'DENY');
  response.headers.set('cache-control', 'no-store');
  return response;
};

export const handleError: HandleServerError = ({ error, event }) => {
  console.error(`${event.request.method} ${event.url.pathname}`, error);
};
