import type { PageServerLoad } from './$types';
import { getNewsList } from '#lib';

export const load: PageServerLoad = async ({ setHeaders }) => {
  try {
    setHeaders({ 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=300' });
  } catch {
    // ignore if header was already set
  }

  const all = await getNewsList({ limit: 40 });

  return {
    timeline: all,
    popular: all.slice(0, 5),
    spotlight: all[1] || all[0] || null,
  };
};
