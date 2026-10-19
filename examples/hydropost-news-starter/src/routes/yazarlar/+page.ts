import type { PageLoad } from './$types';
import { getColumnists } from '#lib';

export const load: PageLoad = async () => {
  const columnists = await getColumnists();
  return { columnists };
};
