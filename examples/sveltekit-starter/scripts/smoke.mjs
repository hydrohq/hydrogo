// Serves the static build/ output the same way the deployed .htaccess does
// (real file -> itself; missing _app/ asset -> 404; anything else -> the
// SPA shell) and checks it actually comes up. Run `npm run build` first.
//
// This starter has no server of its own at runtime (adapter-static), so
// unlike the old adapter-node smoke test, this cannot check a dynamic
// content error turning into a real HTTP 404/502 -- that logic now runs
// client-side in the browser via SvelteKit's error boundary, not as a
// server response status. Checking the static-serving contract itself is
// what's left to verify here.
import { createServer } from 'node:http';
import { createReadStream, existsSync, statSync } from 'node:fs';
import { extname, join } from 'node:path';
import { once } from 'node:events';
import assert from 'node:assert/strict';

const root = new URL('../build/', import.meta.url).pathname.replace(/^\/([a-zA-Z]):/, '$1:');

const MIME = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.xml': 'application/xml', '.txt': 'text/plain', '.svg': 'image/svg+xml', '.ico': 'image/x-icon' };

const server = createServer((req, res) => {
  const path = decodeURIComponent(req.url.split('?')[0]);
  const filePath = join(root, path);
  if (existsSync(filePath) && statSync(filePath).isFile()) {
    res.setHeader('content-type', MIME[extname(filePath)] ?? 'application/octet-stream');
    createReadStream(filePath).pipe(res);
    return;
  }
  if (path.startsWith('/_app/')) {
    res.statusCode = 404;
    res.end('not found');
    return;
  }
  res.setHeader('content-type', 'text/html');
  createReadStream(join(root, 'index.html')).pipe(res);
});

server.listen(0, '127.0.0.1');
await once(server, 'listening');
const port = server.address().port;

const get = async (path) => {
  const res = await fetch(`http://127.0.0.1:${port}${path}`);
  return { status: res.status, body: await res.text() };
};

try {
  const home = await get('/');
  assert.equal(home.status, 200, 'home serves the SPA shell');
  assert.match(home.body, /<title>/);

  assert.equal((await get('/robots.txt')).status, 200, 'robots.txt was prerendered into the build');
  assert.equal((await get('/sitemap.xml')).status, 200, 'sitemap.xml was prerendered into the build');
  assert.equal((await get('/rss.xml')).status, 200, 'rss.xml was prerendered into the build');

  assert.equal((await get('/news/some-article-that-does-not-exist')).status, 200, 'an unknown content path still gets the SPA shell (client-side error boundary takes over from here)');
  assert.equal((await get('/_app/does-not-exist.js')).status, 404, '_app/ never falls back to the SPA shell');

  console.log('starter smoke test passed');
} finally {
  server.close();
}
