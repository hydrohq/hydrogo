import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { getOrder } from '$lib/server/commerce';

export const load: PageServerLoad = async ({ params }) => {
  const order = await getOrder(params.id);
  if (!order) error(404, 'Sipariş bulunamadı');
  return { order };
};
