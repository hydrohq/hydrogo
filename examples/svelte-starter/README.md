# HydroGo Svelte Starter

A modern Svelte 5 + Tailwind CSS starter application consuming the HydroGo v2 Headless REST API.

## Features

- **Pure Headless:** Connects directly to `/api/v1/pages` and `/api/v1/news` endpoints.
- **Svelte 5 Runes:** Written with reactive `$state` and modern component patterns.
- **Dynamic Content:** Renders published pages, block payloads, taxonomy tags, and news articles.
- **Zero Server HTML:** Completely independent of Go templates or server-side markup.

## Getting Started

1. Ensure HydroGo backend is running (e.g. `go run ./cmd/hydro serve` on `http://127.0.0.1:8090`).
2. Run the development server:

```bash
cd examples/svelte-starter
npm install
npm run dev
```

The frontend will be available at `http://localhost:3000` and automatically proxies `/api` requests to the HydroGo backend.
