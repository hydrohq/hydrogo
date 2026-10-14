# Frontend architecture decision guide

HydroGo has one semantic contract and three delivery choices:

| Need | Choose | Boundary |
|---|---|---|
| Public content, Admin Studio, SEO, or a fast prototype | Server-rendered HTML (Templ) | Core owns routing, auth, loading/error states, and safe view models |
| Small partial updates without a full app build | HTMX-style fragment requests | The same server-rendered templates and CSRF/auth boundary; fragments never become a second data API |
| Independent product UI, mobile/web clients, or a separately deployed team | Separate frontend consuming `/api/v1` | SDK/API manifest, CORS/session policy, and version matrix are required |

Start server-rendered unless independent deployment or client-side interaction
complexity is demonstrated. Introduce fragments only for measured interaction
hotspots. Choose a separate frontend when its deployment, ownership, and API
compatibility costs are accepted by the product team.

All choices share semantic data, authorization, locale, error, and loading
contracts. No frontend mode may bypass Core policies or serialize raw database,
cache, queue, or secret details.
