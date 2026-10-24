import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { checkout, CommerceAPIError, getCheckoutConfig, SECURE_COOKIES } from '$lib/server/commerce';
import { itemCatalogData, loadCartLines } from '$lib/server/cartLines';
import { currentMember } from '$lib/server/member';

const IDEMPOTENCY_COOKIE = 'checkout_idem';

export const load: PageServerLoad = async ({ cookies }) => {
  const [{ lines, totalCents }, checkoutConfig, member] = await Promise.all([loadCartLines(cookies), getCheckoutConfig(), currentMember(cookies)]);
  if (lines.length === 0) redirect(303, '/sepet');
  // One key per checkout attempt on this cart, not per click: a double
  // submit (slow network, an impatient second click) must retry the same
  // order, not create two. Cleared once the attempt actually succeeds, in
  // the action below.
  let idempotencyKey = cookies.get(IDEMPOTENCY_COOKIE);
  if (!idempotencyKey) {
    idempotencyKey = crypto.randomUUID();
    cookies.set(IDEMPOTENCY_COOKIE, idempotencyKey, { path: '/', httpOnly: true, sameSite: 'lax', secure: SECURE_COOKIES, maxAge: 60 * 60 });
  }
  return { lines, totalCents, currency: 'TRY', checkoutConfig, memberEmail: member?.email ?? '' };
};

export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const form = await request.formData();
    const guestName = String(form.get('guest_name') ?? '').trim();
    const guestEmail = String(form.get('guest_email') ?? '').trim();
    const guestPhone = String(form.get('guest_phone') ?? '').trim();
    const guestAddress = String(form.get('guest_address') ?? '').trim();
    const guestCity = String(form.get('guest_city') ?? '').trim();
    const guestCountry = String(form.get('guest_country') ?? '').trim();
    const guestPostalCode = String(form.get('guest_postal_code') ?? '').trim();
    const discountCode = String(form.get('discount_code') ?? '').trim();

    const fieldValues = { guestName, guestEmail, guestPhone, guestAddress, guestCity, guestCountry, guestPostalCode, discountCode };
    if (!guestEmail || !guestName || !guestAddress || !guestCity) {
      return fail(400, { message: 'Lütfen ad, e-posta, adres ve şehir alanlarını doldurun.', ...fieldValues });
    }

    const idempotencyKey = cookies.get(IDEMPOTENCY_COOKIE) ?? crypto.randomUUID();
    const { lines } = await loadCartLines(cookies);
    const { itemTitles, itemSKUs } = itemCatalogData(lines);
    try {
      const order = await checkout(cookies, {
        idempotencyKey,
        guestName,
        guestEmail,
        guestPhone,
        guestAddress,
        guestCity,
        guestCountry,
        guestPostalCode,
        discountCode,
        itemTitles,
        itemSKUs
      });
      cookies.delete(IDEMPOTENCY_COOKIE, { path: '/', secure: SECURE_COOKIES });
      if (order.payment?.redirect_url) redirect(303, order.payment.redirect_url);
      redirect(303, `/siparis/${order.id}`);
    } catch (err) {
      if (err instanceof CommerceAPIError) {
        if (err.status === 409) return fail(409, { message: 'Sepetinizdeki bir ürün artık yeterli miktarda stokta değil.', ...fieldValues });
        if (discountCode && err.message.toLowerCase().includes('discount')) {
          return fail(422, { message: 'Girdiğiniz indirim kodu geçerli değil veya artık kullanılamıyor.', ...fieldValues });
        }
        return fail(422, { message: err.message || 'Sipariş tamamlanamadı, lütfen bilgilerinizi kontrol edin.', ...fieldValues });
      }
      throw err;
    }
  }
};
