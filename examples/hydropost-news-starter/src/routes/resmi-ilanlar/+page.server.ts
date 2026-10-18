import type { PageServerLoad } from './$types';
import { getOfficialAnnouncements } from '#lib';

export const load: PageServerLoad = async ({ setHeaders, url }) => {
  setHeaders({
    'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=300',
  });

  const filterType = url.searchParams.get('tur') || undefined;
  const announcements = await getOfficialAnnouncements(50);

  const filtered = filterType
    ? announcements.filter((a) => a.announcement_type === filterType)
    : announcements;

  return {
    announcements: filtered,
    currentFilter: filterType || 'all',
  };
};
