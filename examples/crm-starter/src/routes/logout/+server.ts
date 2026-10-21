import { redirect } from '@sveltejs/kit';
import { logout } from '$lib/server/hydro';
import { destroySession } from '$lib/server/sessions';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ locals, cookies }) => {
  if (locals.session) await logout(locals.session);
  destroySession(locals.sid ?? undefined);
  cookies.delete('crm_sid', { path: '/' });
  redirect(303, '/login');
};
