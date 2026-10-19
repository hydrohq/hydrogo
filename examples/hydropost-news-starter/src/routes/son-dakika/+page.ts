import type { PageLoad } from './$types';
import { getNewsList } from '#lib';

export const load: PageLoad = async () => {
  const all = await getNewsList({ limit: 40 });

  return {
    timeline: all,
    popular: all.slice(0, 5),
  };
};
