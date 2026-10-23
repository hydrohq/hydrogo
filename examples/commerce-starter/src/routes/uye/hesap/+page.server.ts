import { redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { currentMember, listMemberOrders, logoutMember } from '$lib/server/member';

export const load: PageServerLoad = async ({ cookies }) => {
  const member = await currentMember(cookies);
  if (!member) redirect(303, '/uye/giris');
  const orders = await listMemberOrders(cookies);
  return { member, orders };
};

export const actions: Actions = {
  logout: async ({ cookies }) => {
    await logoutMember(cookies);
    redirect(303, '/');
  }
};
