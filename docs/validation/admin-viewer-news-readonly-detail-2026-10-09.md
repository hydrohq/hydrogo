# Viewer read-only news detail — 2026-10-09

The authenticated Editor browser flow creates a draft news item. The Viewer
then opens the same record at `/admin/news/{id}` and sees its title and body
in the read-only detail view, with no mutation form. A direct update request
with the Viewer session’s valid CSRF token returns 403. Reopening the record
shows the Editor’s original title, proving the denied request did not change
the saved record.

The focused `editor and viewer sessions enforce the Edition role boundary`
Chromium scenario passed **1/1 on SQLite** and **1/1 on PostgreSQL 18.6**. The
PostgreSQL run used the local `arda` role (`NOSUPERUSER NOBYPASSRLS`) and a
disposable database that was dropped after the test.

This proves one default Viewer draft-detail read and update-denial browser
path. News publication/scheduling/retraction, role overrides across the full
News route set, deployed Edition identity, and the complete Admin route × role
× CSRF matrix remain open.
