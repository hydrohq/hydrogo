import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    sse_saturation: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: '5s', target: 500 },
        { duration: '10s', target: 1000 },
        { duration: '5s', target: 0 },
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.02'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8090';

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/events`, {
    headers: {
      'Accept': 'text/event-stream',
    },
    timeout: '3s',
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
  });
  sleep(1);
}
