import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ url }) => {
  const body = `User-agent: *
Disallow: /admin/
Disallow: /api/
Disallow: /auth/
Disallow: /csrf-token
Disallow: /yonetim/

Sitemap: ${url.origin}/sitemap.xml
Sitemap: ${url.origin}/sitemap-news.xml
`;

  return new Response(body, {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'public, max-age=3600',
    },
  });
};
