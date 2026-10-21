import { env } from '$env/dynamic/private';
import { fail, redirect } from '@sveltejs/kit';
import { login } from '$lib/server/hydro';
import { createSession, sessionTtlSeconds } from '$lib/server/sessions';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = ({ locals }) => {
  if (locals.session) redirect(303, '/');
};

export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const form = await request.formData();
    const email = String(form.get('email') ?? '').trim();
    const password = String(form.get('password') ?? '');
    const mfa = String(form.get('mfa_code') ?? '').trim();
    if (!email || !password) return fail(400, { error: 'invalid', email, mfa: false });
    let result;
    try {
      result = await login(email, password, mfa);
    } catch {
      return fail(502, { error: 'unavailable', email, mfa: false });
    }
    if (!result.ok) return fail(result.mfaRequired ? 401 : 400, { error: result.message, email, mfa: !!result.mfaRequired });
    const sid = createSession({ cookie: result.cookie, csrf: result.csrf, email });
    cookies.set('crm_sid', sid, {
      path: '/',
      httpOnly: true,
      sameSite: 'lax',
      secure: env.CRM_SECURE_COOKIES === 'true',
      maxAge: sessionTtlSeconds
    });
    redirect(303, '/');
  }
};
