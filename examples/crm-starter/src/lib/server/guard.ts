import { fail, redirect, type Cookies } from '@sveltejs/kit';
import { HydroError } from './hydro';
import { destroySession } from './sessions';

/**
 * Turn an API error into a form failure the page can show, with the request id.
 * A 401 means HydroGo ended the session, so send the user to sign in again.
 */
export function failure(error: unknown, cookies: Cookies, sid: string | null) {
  if (error instanceof HydroError) {
    if (error.status === 401) {
      destroySession(sid ?? undefined);
      cookies.delete('crm_sid', { path: '/' });
      redirect(303, '/login');
    }
    return fail(error.status === 404 ? 404 : error.status >= 500 ? 502 : error.status === 409 ? 409 : 400, {
      error: error.status === 409 ? 'conflict' : 'api',
      message: error.message,
      requestId: error.requestId,
      fields: error.fields ?? {}
    });
  }
  throw error;
}

/** Load-time errors: same 401 handling, other failures bubble to +error. */
export function guardLoad(error: unknown, cookies: Cookies, sid: string | null): never {
  if (error instanceof HydroError && error.status === 401) {
    destroySession(sid ?? undefined);
    cookies.delete('crm_sid', { path: '/' });
    redirect(303, '/login');
  }
  throw error;
}

/** Read text fields from a form, trimmed; empty values are omitted. */
export function textFields(form: FormData, names: string[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const name of names) {
    const value = form.get(name);
    if (typeof value === 'string' && value.trim() !== '') out[name] = value.trim();
  }
  return out;
}
