# Generated draft inbox browser locale acceptance — 2026-10-10

The existing generated editorial browser journey now checks the draft inbox in
English before switching to Turkish for the review and publish workflow. The
English assertions cover the localized page heading, table headers, explicit
open action, draft status label, and axe WCAG 2.1 A/AA results. Existing Turkish
assertions continue to cover the same table structure and the full working-copy
review/publish flow.

Validation passed:

```sh
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/adminstudio \
  -run TestGeneratedAdminDraftsStayIsolatedAndUseTwoVersionCAS -count=1
GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./internal/adminstudio
cd e2e && ./node_modules/.bin/playwright test custom/generated-richtext.spec.ts --list
git diff --check
```

The focused browser journey was attempted but failed before fixture startup
because the environment denies binding `127.0.0.1` (`listen EPERM`). Thus the
new English browser assertions and axe run are discovered/type-checked, but not
runtime verified in this environment.
