import type { LayoutServerLoad } from './$types';
import { getMasthead } from '#lib';

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

  return {
    whatsappPhone: phone,
  };
};
