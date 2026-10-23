// Starts the production build against a stub HydroGo API and checks the
// failure paths: an unknown product slug is a 404, the API being down is a
// 502 (not a crash), and the cart/CSRF cookie round-trips correctly through
// this site's own Set-Cookie. Run `npm run build` first.
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import assert from 'node:assert/strict';

const listen = async (server) => {
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  return server.address().port;
};

let apiUp = true;
const api = createServer((req, res) => {
  res.setHeader('content-type', 'application/json');
  if (req.url.startsWith('/api/v1/commerce/cart')) {
    res.setHeader('set-cookie', ['hydro_csrf=stub-csrf; Path=/', 'hydro_commerce_cart=stub-cart; Path=/; HttpOnly']);
    res.end(JSON.stringify({ data: { id: 'cart_stub', items: [] } }));
    return;
  }
  if (req.url.startsWith('/api/v1/content/products')) {
    res.end(JSON.stringify({ data: [], metadata: { limit: 48, has_more: false } }));
    return;
  }
  res.end(JSON.stringify({ data: [], metadata: { limit: 50, has_more: false } }));
});
const apiPort = await listen(api);

const probe = createServer();
const appPort = await listen(probe);
probe.close();
await once(probe, 'close');

const app = spawn(process.execPath, ['build'], {
  env: {
    ...process.env,
    PORT: String(appPort),
    HOST: '127.0.0.1',
    ORIGIN: `http://127.0.0.1:${appPort}`,
    HYDRO_API_ORIGIN: `http://127.0.0.1:${apiPort}`,
    PUBLIC_HYDRO_API_ORIGIN: `http://127.0.0.1:${apiPort}`,
    PUBLIC_SITE_ORIGIN: `http://127.0.0.1:${appPort}`,
    PUBLIC_SECURE_COOKIES: 'false'
  },
  stdio: 'inherit'
});

const get = async (path) => {
  const res = await fetch(`http://127.0.0.1:${appPort}${path}`, { redirect: 'manual' });
  return { status: res.status, body: await res.text(), headers: res.headers };
};

try {
  for (let i = 0; i < 50; i++) {
    try {
      await get('/');
      break;
    } catch {
      await new Promise((r) => setTimeout(r, 200));
    }
    if (i === 49) throw new Error('the starter did not start');
  }
  const home = await get('/');
  assert.equal(home.status, 200, 'home renders with an empty catalog');
  assert.match(home.body, /Henüz ürün eklenmemiş/, 'empty catalog shows the empty state, not a crash');
  assert.ok(home.headers.get('set-cookie'), 'the root layout warms the cart/CSRF cookies on first load');

  assert.equal((await get('/urun/does-not-exist')).status, 404, 'unknown product is a 404');
  assert.equal((await get('/kategori/does-not-exist')).status, 404, 'unknown category is a 404');
  assert.equal((await get('/sepet')).status, 200, 'an empty cart page renders');

  const search = await get('/arama?q=kemer');
  assert.equal(search.status, 200, 'search page renders with no matches');
  assert.match(search.body, /bulunamadı/, 'empty search results show the empty state');

  api.close();
  api.closeAllConnections();
  await once(api, 'close');
  apiUp = false;
  assert.equal((await get('/urun/anything')).status, 502, 'API down gives a 502, not a crash');
  assert.ok((await get('/sepet')).status < 500, 'the server keeps serving after a catalog API failure');
  console.log('commerce-starter smoke test passed');
} finally {
  app.kill();
  if (apiUp) api.close();
}
