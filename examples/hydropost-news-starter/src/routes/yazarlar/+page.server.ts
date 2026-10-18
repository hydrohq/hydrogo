import type { PageServerLoad } from './$types';
import { getColumnists } from '#lib';

export const load: PageServerLoad = async () => {
  const columnists = await getColumnists();
  return { columnists };
};
