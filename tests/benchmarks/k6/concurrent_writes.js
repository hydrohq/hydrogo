import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    concurrent_writers: {
      executor: 'constant-vus',
      vus: 100,
      duration: '15s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<250'], // 95% of writes under 250ms under concurrency
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8090';
const TOKEN = __ENV.AUTH_TOKEN || '';

export default function () {
  const payload = JSON.stringify({
    title: `Concurrent Post ${__VU}-${__ITER}`,
    content: 'Automated performance benchmark payload',
    status: 'draft',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${TOKEN}`,
    },
  };

  const res = http.post(`${BASE_URL}/api/v1/admin/content/articles`, payload, params);
  check(res, {
    'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
  });
}
