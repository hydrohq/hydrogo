# News taxonomy role and persistence matrix — 2026-10-09

The authenticated Editor opens News taxonomy management and, with its valid
CSRF token, creates a tag, updates it to a category with a new slug, and
deletes it. Each redirect and a fresh taxonomy list read confirm the
operation. The Editor then creates a separate protected term.

The default Viewer is denied the taxonomy management deep link. Update and
delete requests with the Viewer’s valid CSRF token both return `403`. A
separate authenticated Admin reads the taxonomy catalog afterward and confirms
the original protected slug remains and the forged replacement is absent.

The focused Chromium role journey passed **1/1 on SQLite** and **1/1 on
PostgreSQL 18.6** with a `NOSUPERUSER NOBYPASSRLS` application role and a
separate backup operator.

This closes News taxonomy Editor allow and default Viewer deny/persisted-state
cells only. Explicit permission overrides, other News endpoints, other
module-owned routes, and production Edition identity remain outside this
slice.
