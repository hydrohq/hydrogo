# HydroGo headless starter

This is an independent TypeScript frontend foundation. It imports no HydroGo Go package and consumes only the versioned `/api/v1` surface.

```powershell
npm install
$env:HYDRO_API_ORIGIN="http://127.0.0.1:8090"
npm run build
npm run demo
```

`src/hydrogo.ts` exposes a typed client for published Pages, products, variants, cart and checkout. A real Next.js, Nuxt or Astro application can import this client and own all rendering, routing, SEO and component decisions.

For authenticated browser calls, pass the CSRF value together with the browser session cookie; `addCartItem`, `removeCartItem` and `checkout` intentionally require it. Cross-origin cookie, CORS and preview-token policy are not guessed by this starter: they are a separate Core security contract in the headless migration plan.
