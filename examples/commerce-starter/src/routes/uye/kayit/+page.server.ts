import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { currentMember, MemberAPIError, registerMember } from '$lib/server/member';

export const load: PageServerLoad = async ({ cookies }) => {
  if (await currentMember(cookies)) redirect(303, '/uye/hesap');
  return {};
};

export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const form = await request.formData();
    const email = String(form.get('email') ?? '').trim();
    const password = String(form.get('password') ?? '');
    const passwordConfirm = String(form.get('password_confirm') ?? '');
    if (!email || !password) {
      return fail(400, { message: 'E-posta ve şifre gerekli.', email });
    }
    if (password !== passwordConfirm) {
      return fail(400, { message: 'Şifreler eşleşmiyor.', email });
    }
    try {
      await registerMember(cookies, email, password);
    } catch (err) {
      if (err instanceof MemberAPIError) {
        if (err.status === 409) return fail(409, { message: 'Bu e-posta zaten kayıtlı.', email });
        return fail(400, { message: err.message || 'Kayıt tamamlanamadı.', email });
      }
      throw err;
    }
    redirect(303, '/uye/hesap');
  }
};
