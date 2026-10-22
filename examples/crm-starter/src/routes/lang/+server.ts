import { redirect } from '@sveltejs/kit';
import { pickLang } from '$lib/i18n';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ request, cookies }) => {
  const form = await request.formData();
  cookies.set('crm_lang', pickLang(String(form.get('lang') ?? '')), { path: '/', maxAge: 60 * 60 * 24 * 365, sameSite: 'lax', httpOnly: true });
  const back = String(form.get('back') ?? '/');
  redirect(303, back.startsWith('/') && !back.startsWith('//') ? back : '/');
};
