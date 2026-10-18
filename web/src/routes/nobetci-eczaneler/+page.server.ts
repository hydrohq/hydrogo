import type { PageServerLoad } from './$types';
import { getDutyPharmacies, getPrayerTimes, getWeather, normalizeCityName } from '#lib';

export const load: PageServerLoad = async ({ setHeaders, url }) => {
  setHeaders({
    'Cache-Control': 'public, s-maxage=300, stale-while-revalidate=900',
  });

  const rawCity = url.searchParams.get('sehir') || 'İstanbul';
  const city = normalizeCityName(rawCity);
  const pharmacies = getDutyPharmacies(city);
  const prayer = getPrayerTimes(city);
  const weather = getWeather(city);

  return {
    city,
    pharmacies,
    prayer,
    weather,
  };
};
