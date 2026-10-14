# Admin permission E2E shared page fixture — 2026-10-10

## Finding and change

The serial Admin management browser tests share one temporary Edition database. The first role-boundary test creates a page ID, and a later Admin permission test uses that persisted record to verify denied update/publish attempts. The ID was scoped inside the first test callback, so the later callback referenced an undeclared variable. Moved the ID to the spec's shared fixture scope; the first test still assigns it from the created record and the later test can use it after the server restart.

## Verification

- Playwright discovery for `custom/admin-management.spec.ts` passed and lists all 9 cases.
- `git diff --check` passed.
- Runtime test is not claimed: the fixture server cannot bind `127.0.0.1` in this environment (`listen EPERM`), before the browser journey begins.
