# R5 configured public home locale validation — 2026-09-14

Related decision: [ADR-0090](../architecture/0090-configured-public-home-locale.md)

- `cmd/hydro/main_test.go` verifies configured `tr-TR`, empty-locale `en`
  fallback, and path escaping.
- `modules/site/model_test.go` verifies that only a public internal
  `site.home_path` overrides the fallback; private, external, network-path,
  and newline values are ignored.
- The root redirect now uses `cfg.I18N.DefaultLocale` instead of a literal
  `tr` segment and consults the Site module setting when enabled. `seed-demo`
  now stores its page/news fixtures with the same configured locale and rewrites
  fixture links accordingly.
- `modules/news/transport/locale_links_test.go` verifies that static Newsroom
  header/footer links are rewritten to the active locale and fall back safely.
- `go test ./cmd/hydro ./modules/site -p 1 -count=1` passed.

The seeded demo path remains the fallback for installations without a public
home setting. Production content selection and browser matrix coverage remain
separate gates.
