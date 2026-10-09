# Viewer News content.write override — 2026-10-09

The live Admin permission form grants the Viewer `admin.content.read` and
`admin.content.write`. After signing in with that account, the Viewer opens
`/admin/news/new`, creates a News draft in the browser, and receives the saved
record URL. The separate Admin session reads the item through `/api/news/{id}`
and confirms its title and `draft` status persisted. The existing flow then
restores both overrides to `inherit` and verifies the Viewer session is
revoked (`/auth/me` returns 401).

During the same journey, the Viewer Pages create form, News create form, and
News edit form are scanned with axe-core for WCAG 2.1 A/AA. All three passed
with zero violations.

The focused Chromium role journey passed **1/1 on SQLite** and **1/1 on
PostgreSQL 18.6** using a `NOSUPERUSER NOBYPASSRLS` application role and a
separate backup operator.

This proves one explicit Viewer write-override through News HTML and the
accessibility of the three exercised forms. It does not cover all News
role/override combinations, API mutations under the override, theme variants,
or production Edition identity/session behavior.
