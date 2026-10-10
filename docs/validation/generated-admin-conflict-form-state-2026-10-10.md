# Generated Admin 409 Conflict form state and value retention — 2026-10-10

## Verified locally

Enhanced `internal/adminstudio/generated_http.go` so that `409 Conflict` (stale version / concurrent editorial update) responses:
- Render the full Admin form with `http.StatusConflict` (instead of throwing a plain-text 409 page).
- Display the localized conflict alert banner (`admin.generated.error.conflict`) in `ValidationSummary`.
- Retain all submitted form values across primitive, text, relation, array, and widget fields.
- Tested in both Turkish and English interface languages.

```sh
go test ./internal/adminstudio -count=1
go vet ./internal/adminstudio
```

Both commands passed.

## Acceptance boundary

The test `TestGeneratedAdminShowsRecoverableConflictMessage` verifies:
- `409 Conflict` status code.
- Presence of localized conflict copy in TR ("Siz düzenlerken bu kayıt değişti. Sayfayı yenileyip tekrar deneyin.") and EN ("This record changed while you were editing it. Reload the page and try again.").
- Retention of user-entered field values (`value="Stale"`) in the form markup.
