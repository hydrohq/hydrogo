import type { PageServerLoad } from './$types';
import { getMasthead } from '#lib';

export const load: PageServerLoad = async ({ setHeaders }) => {
  setHeaders({
    'Cache-Control': 'public, s-maxage=300, stale-while-revalidate=600',
  });

  const masthead = await getMasthead();

  return {
    masthead: masthead || {
      site_name: 'HydroPost Haber Portalı',
      publisher: 'HydroPost Medya ve Yayıncılık A.Ş.',
      managing_editor: 'Genel Yayın Yönetmeni',
      news_director: 'Haber Müdürü',
      legal_counsel: 'Av. Hukuk Müşavirliği',
      hosting_provider: 'HydroGo Single-Binary Cloud Hosting',
      address: 'Merkez Mah. Basın Cad. No:1, İstanbul',
      phone: '+90 (212) 555 01 00',
      email: 'iletisim@hydropost.com',
      kep_address: 'hydropost@hs01.kep.tr',
      publication_type: 'İnternet Haber Sitesi (Yerel Süreli)',
      editorial_principles: '<p>Yayınlarımız Basın Meslek İlkelerine ve Türkiye Cumhuriyeti Anayasasına tam uyumlu olarak gerçekleştirilmektedir.</p>',
    },
  };
};
