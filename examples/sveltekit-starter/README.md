# HydroGo SvelteKit starter (server-rendered)

A small public site on the HydroGo API: home, news list, news article, pages by slug, and `sitemap.xml`. Pages are **rendered on the server** (adapter-node), so search engines and link-preview bots get full HTML with titles, canonical links, Open Graph tags and JSON-LD, and a publish in the admin is live on the next request. No rebuild on publish.

```bash
cp .env.example .env     # point HYDRO_API_ORIGIN at your HydroGo
npm install
npm run dev              # http://localhost:5173
npm run build && npm start   # production server on :3000
```

It uses [`@hydrogo/client`](../../packages/client). Inside this repository, until the package is published, install it from the sibling folder:
`npm install ../../packages/client`.

This is a starting point, not a theme: it has intentionally almost no styling. Replace the markup, keep the `+page.server.ts` loaders.
