import type { Handle, HandleServerError } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  const response = await resolve(event);
  response.headers.set('x-content-type-options', 'nosniff');
  response.headers.set('referrer-policy', 'strict-origin-when-cross-origin');
  return response;
};

export const handleError: HandleServerError = ({ error, event }) => {
  console.error(`${event.request.method} ${event.url.pathname}`, error);
};
