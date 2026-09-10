# Admin News frontend comparison

This is the HTMX vertical slice for the Admin Studio News experience.
It consumes the HydroGo news surface:

- HTMX: server-rendered table fragment via `/admin/news?fragment=table`.

Run either prototype with a local Vite server and proxy `/admin` and `/api` to
`http://127.0.0.1:8090`. The backend must already be running and the browser
must have an authenticated admin session. This keeps the experiment honest:
the frontend choice does not change authorization or domain behavior.

The first comparison measures list loading, search, detail modal ergonomics,
loading/error states, and visual consistency. Mutations and the block editor
are deliberately the next gate, not mocked into this slice.

## HTMX

```text
cd examples/admin-news-compare/htmx
npx vite --host 127.0.0.1 --port 4173
```
