# HydroGo headless starter

This minimal starter consumes the versioned Pages API without importing HydroGo internals.

```ts
import { getPage } from "./src/hydrogo";

const page = await getPage("en", "home");
console.log(page.title, page.blocks);
```

Configure `HYDRO_API_ORIGIN` for the deployment. The public HTML alternative is `/public/pages/{locale}/{slug}/html`; headless clients should use `/api/v1/pages/...` and render only the published projection.
