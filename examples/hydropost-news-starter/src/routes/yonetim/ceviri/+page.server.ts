import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

const PANEL_COOKIE = 'hp_ceviri_panel';
const PANEL_PASSWORD = process.env.TRANSLATE_PANEL_PASSWORD;

function isAuthed(cookies: { get: (name: string) => string | undefined }): boolean {
  if (!PANEL_PASSWORD) return false;
  return cookies.get(PANEL_COOKIE) === PANEL_PASSWORD;
}

export const load: PageServerLoad = async ({ cookies }) => {
  return {
    configured: Boolean(PANEL_PASSWORD),
    authed: isAuthed(cookies),
  };
};

export const actions: Actions = {
  login: async ({ request, cookies }) => {
    if (!PANEL_PASSWORD) {
      return fail(500, { error: 'TRANSLATE_PANEL_PASSWORD is not configured on the server.' });
    }
    const form = await request.formData();
    const password = String(form.get('password') || '');
    if (password !== PANEL_PASSWORD) {
      return fail(401, { error: 'Wrong password.' });
    }
    cookies.set(PANEL_COOKIE, password, { path: '/yonetim/ceviri', httpOnly: true, sameSite: 'strict' });
    return { success: true };
  },

  logout: async ({ cookies }) => {
    cookies.delete(PANEL_COOKIE, { path: '/yonetim/ceviri' });
    throw redirect(303, '/yonetim/ceviri');
  },

  translate: async ({ request, cookies, fetch }) => {
    if (!isAuthed(cookies)) {
      return fail(401, { error: 'Not authenticated.' });
    }
    const form = await request.formData();
    const id = String(form.get('id') || '').trim();
    const targetLocale = String(form.get('targetLocale') || 'en').trim();
    if (!id) {
      return fail(400, { error: 'News item ID is required.' });
    }

    const apiSecret = process.env.TRANSLATE_API_SECRET || process.env.HYDROGO_ADMIN_API_KEY || '';
    const res = await fetch('/api/translate', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${apiSecret}`,
      },
      body: JSON.stringify({ id, targetLocale }),
    });

    const body = await res.json().catch(() => ({ error: 'Invalid response from translate endpoint.' }));
    if (!res.ok) {
      return fail(res.status, { error: body.error || 'Translation failed.', details: body.details });
    }
    return { success: true, result: body };
  },
};
