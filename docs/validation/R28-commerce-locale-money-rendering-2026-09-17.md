# R28 — Locale-aware commerce money rendering (2026-09-17)

Commerce HTML surfaces now render integer-cent values through the shared
locale formatter instead of legacy float-style templates. Product, variant,
catalog, cart, checkout and payment-summary views use exact integer-cent
formatting with locale grouping and decimal separators. The formatter handles
`MinInt64`/`MaxInt64` ranges without a `float64` conversion.

Verification: `go test ./internal/i18n ./modules/commerce/transport -count=1` —
PASS. The shared formatter tests cover Turkish separators, English grouping and
the maximum signed cent value. Provider payload formatting remains in the
commerce money boundary and is unchanged.
