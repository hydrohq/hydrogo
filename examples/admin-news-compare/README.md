# Admin News frontend comparison

This is the first vertical slice for choosing the Admin Studio UI model.
Both prototypes consume the same HydroGo news surface:

- HTMX: server-rendered table fragment via `/admin/news?fragment=table`.
- React: JSON read model via `/api/v1/news?locale=tr&limit=50`.

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

## React

```text
cd examples/admin-news-compare/react
npm install
npm run dev -- --host 127.0.0.1 --port 4174
```
