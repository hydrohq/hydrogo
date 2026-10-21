import { env } from '$env/dynamic/private';
import { redirect, type Handle, type HandleServerError } from '@sveltejs/kit';
import { pickLang } from '$lib/i18n';
import { getSession } from '$lib/server/sessions';

export const SESSION_COOKIE = 'crm_sid';
const PUBLIC_PATHS = new Set(['/login', '/health']);

export const handle: Handle = async ({ event, resolve }) => {
  const sid = event.cookies.get(SESSION_COOKIE) ?? null;
  event.locals.sid = sid;
  event.locals.session = getSession(sid ?? undefined);
  event.locals.lang = pickLang(event.cookies.get('crm_lang'), pickLang(env.CRM_LANG));

  if (!event.locals.session && !PUBLIC_PATHS.has(event.url.pathname)) {
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
