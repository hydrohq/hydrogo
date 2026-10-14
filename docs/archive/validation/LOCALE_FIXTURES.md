# Locale and direction validation

HydroGo keeps locale-neutral values in the domain and formats them only at
render time. The core contract is covered by `internal/i18n` unit tests and
the opt-in Playwright fixtures in `e2e/locale.spec.ts`.

## Core contract

The unit suite verifies Turkish, English, and Japanese date/number output,
Turkish and English currency output, invalid-currency fail-closed behavior,
and safe fallback for malformed locale tags:

```powershell
go test ./internal/i18n
```

## Browser contract

The browser suite verifies that:

- Turkish documents are explicitly `lang=tr` and `dir=ltr`.
- Arabic documents are explicitly `dir=rtl` and do not overflow horizontally.
- Long mixed-direction headings remain within the viewport.

Run it only against an isolated seeded server:

```powershell
$env:HYDRO_VISUAL = "1"
npx playwright test e2e/locale.spec.ts
```

The browser fixture is intentionally not marked as a production deployment
claim: it requires a running application and seeded content. Product teams
remain responsible for translated copy, pluralization review, and the complete
locale matrix for each Client Edition.
