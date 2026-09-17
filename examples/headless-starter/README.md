# HydroGo headless starter

This is an independent TypeScript frontend foundation. It imports no HydroGo Go package and consumes only the versioned `/api/v1` surface.

```powershell
npm install
$env:HYDRO_API_ORIGIN="http://127.0.0.1:8090"
npm run build
npm run demo
```

`src/hydrogo.ts` exposes a typed client for published Pages and News (listing and detail), products, variants, cart and checkout. A real Next.js, Nuxt or Astro application can import this client and own all rendering, routing, SEO and component decisions.

Client requests use a 15-second default timeout (configurable up to 120 seconds
with `timeoutMs`) and cap error response text at 8,192 code points. Handle
`AbortError`/`HydroGoError` in the host application and show audience-appropriate
copy rather than exposing raw infrastructure details. `HydroGoError` exposes a
bounded `code`, `message` and optional `requestId`; raw response text is not
part of the public error contract.

For route/lifecycle cancellation, pass an `AbortSignal` in the client options;
the signal is forwarded to every request and is combined with the bounded
timeout controller.

`checkout` validates the caller-supplied idempotency key (16–128 visible ASCII
characters) before sending it. Reuse the same key only when retrying the same
logical checkout.

For authenticated browser calls, pass the CSRF value together with the browser session cookie; `addCartItem`, `removeCartItem` and `checkout` intentionally require it. Cross-origin cookie, CORS and preview-token policy are not guessed by this starter: they are a separate Core security contract in the headless migration plan.
