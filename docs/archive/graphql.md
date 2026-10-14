# Headless GraphQL API

HydroGo exposes a public, read-only GraphQL endpoint at `POST /api/graphql` (and the versioned alias `POST /api/v1/graphql`). Mutations remain on the versioned REST endpoints so CSRF, idempotency, and audit rules have one write path.

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

The current Site Edition publishes Pages and News read models only. Commerce is deferred and has no GraphQL surface in this release.
