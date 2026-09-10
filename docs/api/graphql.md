# Headless GraphQL API

HydroGo exposes a public, read-only GraphQL endpoint at `POST /api/graphql` (and the versioned alias `POST /api/v1/graphql`). Mutations remain on the versioned REST endpoints so CSRF, idempotency, audit, and payment authorization rules have one write path.

Example query:

```graphql
query PublishedPage($locale: String!, $slug: String!) {
  page(locale: $locale, slug: $slug) {
    id
    locale
    slug
    title
    body
    blocks { type payload }
  }
}
```

Commerce-enabled instances additionally expose `products` for published catalog items. The endpoint uses the same published-page cache and catalog service contracts as the REST/HTML transports; it does not read SQL directly.
