import type { Handle, HandleServerError } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  // GHSA-pfhg-q3qv-9q6r (CVE-2026-114060): @sveltejs/kit < 3.0.0 can turn a
  // `//`-prefixed action path into an open redirect through use:enhance.
  // No catch-all route accepts one here, but this is the SvelteKit security
  // team's own documented workaround -- reject it outright before route
  // resolution rather than rely on that absence staying true forever.
  if (event.url.pathname.startsWith('//')) {
    return new Response('Not found', { status: 404 });
  }
  const response = await resolve(event, { transformPageChunk: ({ html }) => html.replace('%lang%', event.locals.lang ?? 'en') });
  response.headers.set('x-content-type-options', 'nosniff');
  response.headers.set('referrer-policy', 'strict-origin-when-cross-origin');
  return response;
};

export const handleError: HandleServerError = ({ error, event }) => {
  console.error(`${event.request.method} ${event.url.pathname}`, error);
};
