import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { currentMember, loginMember, MemberAPIError } from '$lib/server/member';

export const load: PageServerLoad = async ({ cookies }) => {
  if (await currentMember(cookies)) redirect(303, '/uye/hesap');
  return {};
};

export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const form = await request.formData();
    const email = String(form.get('email') ?? '').trim();
    const password = String(form.get('password') ?? '');
    if (!email || !password) {
      return fail(400, { message: 'E-posta ve şifre gerekli.', email });
    }
    try {
      await loginMember(cookies, email, password);
    } catch (err) {
      if (err instanceof MemberAPIError) {
        return fail(err.status === 429 ? 429 : 401, { message: 'E-posta veya şifre yanlış.', email });
      }
      throw err;
    }
    redirect(303, '/uye/hesap');
  }
};
