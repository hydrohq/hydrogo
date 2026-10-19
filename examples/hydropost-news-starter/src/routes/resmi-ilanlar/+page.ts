import type { PageLoad } from './$types';
import { getOfficialAnnouncements } from '#lib';

export const load: PageLoad = async ({ url }) => {
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
