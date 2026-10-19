// Starts the production build against a stub HydroGo API and checks the
// failure paths: content pages answer 404 for unknown slugs, 502 when the API
// is down, and the server stays up. Run `npm run build` first.
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
  if (req.url.includes('/by-slug')) {
    res.statusCode = 404;
    res.end(JSON.stringify({ code: 'not_found', message: 'not found', request_id: 'smoke' }));
    return;
  }
  res.end(JSON.stringify({ data: [], metadata: { total: 0 } }));
});
const apiPort = await listen(api);

const probe = createServer();
const appPort = await listen(probe);
probe.close();
await once(probe, 'close');

const app = spawn(process.execPath, ['build'], {
  env: { ...process.env, PORT: String(appPort), HOST: '127.0.0.1', ORIGIN: `http://127.0.0.1:${appPort}`, HYDRO_API_ORIGIN: `http://127.0.0.1:${apiPort}`, PUBLIC_SITE_ORIGIN: `http://127.0.0.1:${appPort}` },
  stdio: 'inherit'
});

const get = async (path) => {
  const res = await fetch(`http://127.0.0.1:${appPort}${path}`, { redirect: 'manual' });
  return { status: res.status, body: await res.text() };
};

try {
  for (let i = 0; i < 50; i++) {
    try { await get('/robots.txt'); break; } catch { await new Promise((r) => setTimeout(r, 200)); }
    if (i === 49) throw new Error('the starter did not start');
  }
  const home = await get('/');
  assert.equal(home.status, 200, 'home renders with an empty API');
  assert.match(home.body, /<title>/);
  assert.equal((await get('/news/does-not-exist')).status, 404, 'unknown article is a 404');
  assert.equal((await get('/sitemap.xml')).status, 200);

  api.close();
  api.closeAllConnections();
  await once(api, 'close');
  apiUp = false;
  const down = await get('/news/some-article');
  assert.equal(down.status, 502, 'API down gives a 502, not a crash');
  assert.ok((await get('/robots.txt')).status === 200, 'the server keeps serving after an API failure');
  console.log('starter smoke test passed');
} finally {
  app.kill();
  if (apiUp) api.close();
}
