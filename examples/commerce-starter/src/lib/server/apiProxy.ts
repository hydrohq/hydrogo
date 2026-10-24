import type { Cookies } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';

export const API_ORIGIN = (env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
export const SECURE_COOKIES = publicEnv.PUBLIC_SECURE_COOKIES === 'true';

// The only cookies this storefront ever needs to carry between the browser
// and HydroGo. Named exactly as HydroGo names them, so a value read from one
// of HydroGo's Set-Cookie headers can be replayed verbatim as this site's
// own cookie of the same name, and forwarded back to HydroGo unchanged on
// the next request -- a proxy, not a session store: there is no privileged
// secret here to keep off the browser (contrast with examples/crm-starter's
// staff session, which does need that indirection). Shared by commerce.ts
// (cart/CSRF) and member.ts (member session) -- both are HydroGo's own
// cookies, proxied the same way regardless of which HydroGo API set them.
const PROXIED_COOKIES = ['hydro_commerce_cart', 'hydro_csrf', 'hydro_member_session'] as const;

function cookieHeader(cookies: Cookies): string {
  return PROXIED_COOKIES.map((name) => [name, cookies.get(name)] as const)
    .filter(([, value]) => value)
    .map(([name, value]) => `${name}=${value}`)
    .join('; ');
}

function applySetCookies(response: Response, cookies: Cookies): void {
  const raw = typeof response.headers.getSetCookie === 'function' ? response.headers.getSetCookie() : [];
  for (const entry of raw) {
    const [pair] = entry.split(';');
    const eq = pair.indexOf('=');
    if (eq < 1) continue;
    const name = pair.slice(0, eq).trim();
    const value = pair.slice(eq + 1).trim();
    if (!(PROXIED_COOKIES as readonly string[]).includes(name)) continue;
    const cleared = /expires=Thu, 01 Jan 1970/i.test(entry) || /max-age=0/i.test(entry) || value === '';
    if (cleared) {
      cookies.delete(name, { path: '/' });
      continue;
    }
    // HttpOnly regardless of how HydroGo set it: nothing here is read by
    // this starter's own client-side JS, every mutation is a server form
    // action, so there is no reason to ever expose these to the browser's
    // script context.
    cookies.set(name, value, { path: '/', httpOnly: true, sameSite: 'lax', secure: SECURE_COOKIES, maxAge: 60 * 60 * 24 * 30 });
  }
}

/** Calls a HydroGo endpoint with the proxied cookies attached and any new
 * Set-Cookie HydroGo returns replayed onto this site's own response.
 * Returns the raw Response -- callers interpret the body themselves, since
 * HydroGo's public, non-admin endpoints don't all use the same envelope
 * (commerce's is {data,...}; members' auth endpoints return the body
 * directly and errors as plain text, not JSON). */
export async function proxiedFetch(cookies: Cookies, path: string, init: RequestInit = {}): Promise<Response> {
  const method = (init.method ?? 'GET').toUpperCase();
  const headers = new Headers(init.headers);
  const forwarded = cookieHeader(cookies);
  if (forwarded) headers.set('Cookie', forwarded);
  if (method !== 'GET' && method !== 'HEAD') {
    const csrf = cookies.get('hydro_csrf');
    if (csrf) headers.set('X-CSRF-Token', csrf);
  }
  const response = await fetch(`${API_ORIGIN}${path}`, { ...init, method, headers });
  applySetCookies(response, cookies);
  return response;
}
