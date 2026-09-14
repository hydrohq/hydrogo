# R5 validation: locale direction

## Local evidence

- `go test ./internal/canonical ./modules/pages/transport ./modules/news/transport`
- Locale direction tests cover Arabic, Persian, Hebrew, Turkish, English,
  unknown, and empty tags.
- Public page and news document roots now emit `dir="ltr"` or `dir="rtl"`.
- v1 theme layouts receive the same bounded direction and default to `ltr` for
  legacy callers.

## Remaining gate

RTL visual review, mixed-direction text, date/number formatting, and translated
copy require a locale/theme browser matrix and product-owned translations.
