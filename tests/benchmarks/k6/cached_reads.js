import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    cached_reads: {
      executor: 'constant-arrival-rate',
      rate: 2000,
      timeUnit: '1s',
      duration: '15s',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<15'], // 95% of requests should be below 15ms
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8090';

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/content/articles`);
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
