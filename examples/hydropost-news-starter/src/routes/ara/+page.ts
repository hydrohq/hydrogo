import type { PageLoad } from './$types';
import { getNewsList } from '#lib';

export const load: PageLoad = async ({ url }) => {
  const q = (url.searchParams.get('q') || '').trim();
  const results = q ? await getNewsList({ search: q, limit: 100 }) : [];

  return { q, results };
};
