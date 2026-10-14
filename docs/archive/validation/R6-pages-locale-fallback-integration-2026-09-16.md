# R6 validation: public page locale fallback

Date: 2026-09-16

## Evidence

- `go test ./modules/pages/service` passes.
- The service test proves `tr-TR` resolves to `tr`, then `en`, and returns the
  published English page when the first candidate is absent.
- Invalid/duplicate locale configuration fails closed with `ErrInvalidPage`.
- Fallback is limited to `sql.ErrNoRows`; other repository errors are not
  masked.

## Remaining gate

Production acceptance still requires multilingual fixture content and route
checks in each supported Client Edition. This local contract test does not
claim external translation or database certification.
