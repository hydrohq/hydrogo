# R78 — Locale and visual browser acceptance (2026-09-19)

## Result

PASS on the isolated seeded SQLite browser run with `HYDRO_VISUAL=1`:

- Turkish LTR document, Arabic RTL document and mixed-direction overflow: 3/3
- visual golden checks across desktop, tablet and mobile for home, news and
  shop: 9/9
- combined with the functional suite and headless client smoke: 47/47
  Playwright tests passed

The approved PNG baselines under `e2e/visual.spec.ts-snapshots/` were refreshed
after visual review because the current public theme/layout had materially
changed from the previous baselines. Snapshot refresh is available only through
the explicit `HYDRO_E2E_UPDATE_SNAPSHOTS=1` opt-in in `scripts/run-e2e.ps1`.

## Boundary

This is an isolated local browser acceptance result. It does not certify
cross-origin BFF session mutations, production TLS/proxy behavior or external
customer capacity.
