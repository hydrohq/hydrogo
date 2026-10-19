# @hydrogo/client

Typed fetch client for the [HydroGo](https://github.com/ozyildirims/hydrogo) REST API. No dependencies; works in browsers, Node 20+, Deno, Bun and edge runtimes.

```ts
import { createHydroGoClient } from "@hydrogo/client";

const hydro = createHydroGoClient({ origin: "https://cms.example.com" });

const page = await hydro.getPage("en", "about");
const posts = await hydro.listNews("en", 10);

// Any content type from content-types/*.yaml
const faq = await hydro.listContent<{ id: string; question: string }>("faq", { limit: 20, search: "refund" });
const one = await hydro.getContent("faq", faq.data[0].id);
```

Errors are `HydroGoError` with `status`, `code`, `message` and `requestId` (quote the request id when reporting a problem). Requests time out after 15 s (`timeoutMs`, max 120 s), response bodies are size-capped, and member methods never send browser cookies.

The client is hand-written for the public surface; the full admin/OpenAPI types for admin tooling come from `hydro sdk generate --lang typescript`.

## Develop

```bash
npm ci && npm test
```

## Publish

This package is private and is never published to npm or any public registry. Frontends use it from the repository: the starters vendor a copy (`examples/sveltekit-starter/src/lib/hydrogo.ts`, kept in sync by a Go test), or install it from a checkout with `npm install ../path/to/hydrogo/packages/client`.
