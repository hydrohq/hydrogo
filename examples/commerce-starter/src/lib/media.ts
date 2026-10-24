import { env } from '$env/dynamic/public';

// Separate from HYDRO_API_ORIGIN: an <img> tag is fetched by the browser
// directly from HydroGo, not proxied through this SvelteKit server, so it
// needs a public, browser-reachable origin -- which may differ from the
// private one this server uses (a reverse proxy, a different hostname).
const origin = (env.PUBLIC_HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080').replace(/\/$/, '');

export function mediaUrl(id: string): string {
  return `${origin}/media/${encodeURIComponent(id)}`;
}
