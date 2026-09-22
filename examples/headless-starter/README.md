# HydroGo headless starter

This is the presentation layer of a Client Edition, deliberately kept outside
HydroGo Core. HydroGo owns the versioned API contracts, authorization and
operational boundaries; this starter owns rendering, routing, SEO and the
component system. It can be replaced by React, Next, Nuxt, Astro, HTMX or a
native client without changing the Core.

This is an independent TypeScript frontend foundation. It imports no HydroGo Go package and consumes only the versioned `/api/v1` surface.

```powershell
npm install
$env:HYDRO_API_ORIGIN="http://127.0.0.1:8090"
npm run build
npm run demo
```

`src/hydrogo.ts` exposes a typed client for published Pages and News (listing and detail), plus an isolated member-bearer client surface for consumer registration, login, session refresh, profile and device management. A real Next.js, Nuxt or Astro application can import this client and own all rendering, routing, SEO and component decisions.

Client requests use a 15-second default timeout (configurable up to 120 seconds
with `timeoutMs`) and cap error response text at 8,192 code points. Handle
`AbortError`/`HydroGoError` in the host application and show audience-appropriate
copy rather than exposing raw infrastructure details. `HydroGoError` exposes a
bounded `code`, `message` and optional `requestId`; raw response text is not
part of the public error contract.

The configured `origin` must be an absolute HTTP(S) URL without embedded
credentials; unsupported schemes and credential-bearing origins fail before a
request is sent.

For route/lifecycle cancellation, pass an `AbortSignal` in the client options
for global cancellation, or as the final options argument on an individual
client method. The signal is combined with the bounded timeout controller.

Commerce is deferred for this release and is intentionally absent from this
starter. Cross-origin cookie, CORS and preview-token policy are not guessed by
the starter; they remain separate Core security contracts in the headless
migration plan.
