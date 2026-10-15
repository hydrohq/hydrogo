# HydroGo v2 Headless API Guide

HydroGo is a single-binary headless CMS. It exposes all content, media, taxonomy, and administrative operations over a unified REST API conforming to the envelope specification in `internal/api`.

## 1. Response Envelope

All JSON endpoints respond with standard envelopes:

### Success Envelope
```json
{
  "data": { ... },
  "meta": {
    "total": 42,
    "offset": 0,
    "limit": 20,
    "has_more": true
  }
}
```

### Error Envelope
```json
{
  "code": "VALIDATION_FAILED",
  "message": "Title is required and must not be empty",
  "request_id": "req_01j7...",
  "fields": {
    "title": "required"
  }
}
```

## 2. Authentication & Credentials

HydroGo supports three authentication methods:

1. **Same-Origin SPA Sessions:**
   - Cookie: `hydro_session` (HTTP-only, Secure, SameSite)
   - CSRF Header: `X-CSRF-Token` (retrieved via `GET /csrf-token`)

2. **Personal Access & Mobile Bearer Tokens:**
   - Header: `Authorization: Bearer <token>`
   - Generated via `GET /api/v1/admin/me/tokens` or staff UI in Admin.

3. **API Keys (Headless Frontend / SSR / SSG):**
   - Header: `X-API-Key: <api-key>`
   - Scoped for read/preview operations from external static site generators (Next.js, Nuxt, Astro, SvelteKit).

## 3. Public Read Endpoints

When `public: true` is configured in content types or core modules:

- `GET /api/v1/content/<content-type-id>`: List published items for custom content types (FAQ, Team, Events, etc.).
- `GET /api/v1/content/<content-type-id>/<id>`: Detail of a published item.
- `GET /api/v1/pages`: List published pages.
- `GET /api/v1/pages/<slug>`: Fetch page content and block payload.
- `GET /api/v1/news`: List published news items.
- `GET /api/v1/news/<slug>`: Fetch published article.
- `GET /media/<id>`: Serve public media assets and image derivatives.

## 4. Draft Previews

To preview unpublished drafts or revisions in a decoupled frontend:

1. Editors generate a preview link in Admin: `POST /api/v1/admin/pages/{id}/preview-link`
2. The endpoint returns a revision-bound signed token:
   `GET /api/v1/pages/{id}/preview?token=<signed-jwt>`
3. The frontend verifies the token with HydroGo and displays the draft content in real-time.

## 5. Declarative Content Types

To register a new content collection (e.g. Products, Testimonials, Team), add a single YAML file in `content-types/team.yaml`:

```yaml
id: team
plural: team
singular: Team Member
label: Team Members
icon: 👥
public: true
drafts: true
fields:
  - name: name
    label: Full Name
    kind: text
    required: true
    list: true
  - name: role
    label: Position
    kind: text
    list: true
  - name: bio
    label: Biography
    kind: richtext
  - name: photo
    label: Photo
    kind: media
```

Upon server restart, the collection automatically gets:
- Full Admin CRUD UI at `#/content/team`
- Public read API at `/api/v1/content/team` (supports pagination, search query `?q=`, and sorting)
- Full revision control and draft workflow.
