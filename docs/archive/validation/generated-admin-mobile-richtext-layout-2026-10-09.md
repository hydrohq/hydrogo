# Generated Admin mobile rich-text layout — 2026-10-09

## Finding and change

Reviewing the 390px generated article form screenshot showed the rich-text
editor left a large blank area before its metrics. The form also contains a
separate block editor, so this pushed later fields and the primary Save action
well below the first editor. The cause was Quill's
`.richtext-field .ql-container.ql-snow` 280px minimum winning over a lower-
specificity mobile override.

The mobile stylesheet now applies a 180px minimum to the snow container and
editable area at widths up to 600px. Desktop sizing is unchanged. The browser
test asserts the editable area remains between 140px and 220px and the complete
rich-text field, including toolbar and metrics, stays at or below 360px.

## Verification

The full generated-richtext Chromium suite passed **8/8** in 29.3 s against a
fresh local synthetic fixture process. The run also covered Editor.js mobile
touch insertion and JSON persistence, all four theme families in light/dark
modes, axe WCAG 2.1 A/AA, conflicts, relation/media, and editorial workflows.

```sh
GOCACHE=/tmp/hydrogo-go-cache \
  HYDRO_E2E_ADDRESS=127.0.0.1:8101 \
  go run ./tests/e2e/support/generatedrichtext

cd e2e
HYDRO_E2E_EXTERNAL_SERVER=1 \
  HYDRO_E2E_BASE_URL=http://127.0.0.1:8101 \
  TMPDIR=/var/tmp/hydro-go-tests \
  npm exec playwright test custom/generated-richtext.spec.ts -- \
  --project=chromium --workers=1 --reporter=line --trace=off

go test ./internal/ui ./internal/adminstudio -count=1
go vet ./internal/ui ./internal/adminstudio
```

The external fixture is in-memory but stateful; start a fresh process for each
full suite replay because the editorial tests create and update seeded records.
The browser run uses emulated 390px touch input, not a physical device or a
deployed Edition. `impeccable detect` reported existing side-border, font, and
layout-transition findings in the large Admin stylesheet; it found no issue in
the added mobile editor rule. Those unrelated existing styles were left outside
this focused fix.
