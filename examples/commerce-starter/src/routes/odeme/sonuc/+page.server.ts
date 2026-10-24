import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// PayTR's own ok_url/fail_url redirect: per its docs this carries no
// reliable payment status, only "processing" -- see
// modules/commerce/providers/paytr's VerifyCallback doc comment. The only
// trustworthy thing here is the order id PayTR echoes back (appended by
// paytr.Provider.StartPayment), so this page does nothing but forward to
// the real, server-verified order status.
export const load: PageServerLoad = async ({ url }) => {
  const orderId = url.searchParams.get('oid');
  redirect(303, orderId ? `/siparis/${orderId}` : '/');
};
