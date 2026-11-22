import http from 'k6/http';
import { check, group, sleep } from 'k6';

/**
 * HydroGo Commerce Ad-Burst Load Test
 *
 * Simulates a sudden ad-traffic spike:
 * - 80% browse/stock checks (GET /api/v1/commerce/checkout-config, /stock/:id)
 * - 20% cart + checkout flow (POST /cart/items -> POST /checkout)
 *
 * Usage:
 *   k6 run -e TARGET_URL=http://127.0.0.1:8080 -e VARIANT_ID=variant-1 commerce_ad_burst.js
 */
export const options = {
  scenarios: {
    ad_burst: {
      executor: 'ramping-arrival-rate',
      startRate: 20,
      timeUnit: '1s',
      preAllocatedVUs: 40,
      maxVUs: 200,
      stages: [
        { target: 100, duration: '10s' },
        { target: 250, duration: '20s' },
        { target: 250, duration: '15s' },
        { target: 0, duration: '5s' },
      ],
    },
  },
  thresholds: {
    'http_req_duration{kind:read}': ['p(95)<50', 'p(99)<120'],
    'http_req_duration{kind:checkout}': ['p(95)<300', 'p(99)<800'],
    'http_req_failed{kind:read}': ['rate<0.01'],
    'checks': ['rate>0.99'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8080';
const VARIANT_ID = __ENV.VARIANT_ID || 'variant-load-1';

export default function () {
  const isBuyer = (__VU + __ITER) % 5 === 0;

  group('storefront_read', () => {
    const cfgRes = http.get(`${BASE_URL}/api/v1/commerce/checkout-config`, {
      tags: { kind: 'read' },
    });
    check(cfgRes, {
      'checkout-config no 5xx': (r) => r.status > 0 && r.status < 500,
    });

    const stockRes = http.get(`${BASE_URL}/api/v1/commerce/stock/${VARIANT_ID}`, {
      tags: { kind: 'read' },
    });
    check(stockRes, {
      'stock no 5xx': (r) => r.status > 0 && r.status < 500,
    });
  });

  if (isBuyer) {
    group('checkout_flow', () => {
      const jar = http.cookieJar();
      jar.clear(BASE_URL);

      const addPayload = JSON.stringify({
        variant_id: VARIANT_ID,
        quantity: 1,
        unit_price_cents: 1500,
      });

      const addRes = http.post(`${BASE_URL}/api/v1/commerce/cart/items`, addPayload, {
        headers: { 'Content-Type': 'application/json' },
        tags: { kind: 'checkout' },
      });

      const addOk = check(addRes, {
        'cart add no 5xx': (r) => r.status > 0 && r.status < 500,
      });

      if (addOk && (addRes.status === 200 || addRes.status === 201)) {
        const idemKey = `k6-${__VU}-${__ITER}-${Date.now()}`;
        const checkoutPayload = JSON.stringify({
          guest_email: `load-${__VU}-${__ITER}@example.com`,
          idempotency_key: idemKey,
        });

        const coRes = http.post(`${BASE_URL}/api/v1/commerce/checkout`, checkoutPayload, {
          headers: { 'Content-Type': 'application/json' },
          tags: { kind: 'checkout' },
        });

        check(coRes, {
          'checkout no 5xx': (r) => r.status > 0 && r.status < 500,
        });
      }
    });
  }

  sleep(0.05);
}
