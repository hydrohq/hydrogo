import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getCurrencies } from '$lib/server/commerce';

const DISPLAY_CURRENCY_COOKIE = 'display_currency';

// A plain link, not a form action: switching the display currency is not
// a cart mutation (it changes nothing about what gets charged -- see
// config.Currencies' own doc comment), so it does not need CSRF
// protection or a POST.
export const GET: RequestHandler = async ({ params, cookies, url }) => {
  const currencies = await getCurrencies();
  const code = params.code;
  if (code === currencies.default || currencies.display.includes(code)) {
    cookies.set(DISPLAY_CURRENCY_COOKIE, code, { path: '/', maxAge: 60 * 60 * 24 * 365 });
  }
  const redirectTo = url.searchParams.get('redirect') || '/';
  redirect(303, redirectTo.startsWith('/') ? redirectTo : '/');
};
