import type { LayoutServerLoad } from './$types';
import { getMasthead, getSiteNavigation, getSiteSettings, getNewsList } from '#lib';

export const load: LayoutServerLoad = async () => {
  const envPhone = (typeof process !== 'undefined' && (process.env?.WHATSAPP_PHONE || process.env?.PUBLIC_WHATSAPP_PHONE)) || '';

  let phone = envPhone;
  if (!phone) {
    try {
      const masthead = await getMasthead();
      if (masthead?.phone) {
        phone = masthead.phone;
      }
    } catch {
      // ignore
    }
  }

  const [headerNav, footerPagesNav, footerSocialNav, siteSettings, breakingNews] = await Promise.all([
    getSiteNavigation('header'),
    getSiteNavigation('footer_sayfalar'),
    getSiteNavigation('footer_sosyal'),
    getSiteSettings(),
    getNewsList({ limit: 8 }),
  ]);

  return {
    whatsappPhone: phone,
    headerNav,
    footerPagesNav,
    footerSocialNav,
    siteSettings,
    breakingNews,
  };
};
