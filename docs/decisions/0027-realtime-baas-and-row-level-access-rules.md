# 0027: Realtime BaaS (PocketBase Parity) and Row-Level Access Rules

- Status: Accepted
- Date: 2026-10-29

## Context

HydroGo originally served custom content types as read-only public endpoints (`GET /api/v1/content/<type>`), with mutations restricted to the internal Admin API. While sufficient for traditional headless CMS websites, modern applications (chat platforms, social feeds, collaborative tools, mobile apps) require Backend-as-a-Service (BaaS) capabilities:
1. Public CRUD endpoints where authenticated members (or public visitors where allowed) can create, update, and delete records.
2. Declarative, row-level access rules evaluated dynamically per record and caller identity (similar to PocketBase API rules).
3. Live pub/sub real-time subscriptions with row-level event filtering so connected clients receive instant updates without polling.

## Decision

Extended the content type system into a full Realtime BaaS:

1. **Declarative Row-Level Access Rules (`access_rules`)**:
   - Added `access_rules:` (`list`, `read`, `create`, `update`, `delete`) to content type YAML definitions.
   - Rules evaluate standard boolean expressions over the caller principal (`@auth.id`, `@auth.role`, `@auth.email`, `@auth.is_admin`, `@auth.is_member`) and record fields (`record.<field>` or `<field>`).
   - Supports keywords: `"public"`, `"admin"`, `"member"`, `"true"`, `"false"`.
   - Admin callers bypass access rules, while anonymous callers default to public read on public types and denied mutations unless explicitly permitted.

2. **Public BaaS CRUD Endpoints**:
   - `POST /api/v1/content/{type}`: Creates records, validating schema and `create` rule.
   - `PUT /api/v1/content/{type}/{id}`: Updates records, verifying the `update` rule on both existing stored state and proposed state.
   - `DELETE /api/v1/content/{type}/{id}`: Deletes records, verifying the `delete` rule.
   - `GET /api/v1/content/{type}` and `{id}`: Filters lists and validates reads dynamically per row.

3. **Realtime Pub/Sub Subscriptions via SSE (`realtime: true`)**:
   - `GET /api/v1/content/{type}/subscribe`: Streams Server-Sent Events for `create`, `update`, and `delete` mutations on topic `content:{type}`.
   - Scaled Realtime Hub capacity to 4096 concurrent subscribers.
   - Enforces per-subscriber row-level event filtering: incoming events are evaluated against the subscriber's read rule before delivery, preventing unauthorized data leakage across client boundaries.

4. **Client SDK Support (`@hydrogo/client`)**:
   - Added `createContent<T>`, `updateContent<T>`, `deleteContent`, and `subscribeContent<T>` with automated SSE lifecycle management.
   - Mirrored changes to vendored client in `examples/sveltekit-starter/src/lib/hydrogo.ts`.

## Consequences

- HydroGo achieves functional parity with PocketBase for application backends while retaining its superior enterprise strengths (PostgreSQL/MySQL support, multi-tenancy, revisions, drafts, and headless CMS features).
- Zero new runtime external dependencies were introduced; expressions are compiled with the existing collections AST parser and evaluated in standard Go.
