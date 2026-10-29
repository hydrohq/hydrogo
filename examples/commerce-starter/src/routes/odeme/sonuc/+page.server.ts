import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// Payment provider return redirect: forwards to the server-verified order status.
export const load: PageServerLoad = async ({ url }) => {
  const orderId = url.searchParams.get('oid');
  redirect(303, orderId ? `/siparis/${orderId}` : '/');
};
