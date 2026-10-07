# Playwright server process lifecycle

**Date:** 2026-10-08  
**Result:** 14/14 Chromium tests passed; no fixture server remained afterward.

Several long-lived Playwright fixtures used `go run` and stopped only the Go
launcher. The compiled server process could outlive that launcher. The shared
`e2e/custom/server-process.ts` helper now starts Unix fixtures in a process
group and terminates the whole group; on Windows it uses `taskkill /T` to stop
the process tree. Direct test binaries can use the same stop helper.

The affected suites are Admin management, dynamic Collections, external Edition
Admin, generated CRUD, and generated rich text.

Validation command:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts custom/dynamic-collections.spec.ts \
  custom/generated-admin-crud.spec.ts custom/generated-richtext.spec.ts \
  custom/external-edition-admin.spec.ts --project chromium --workers=1 --reporter=line
```

All five fixtures completed, covering 14 tests. A process-table check after the
run found no new HydroGo or generated Edition test server processes. The
Windows `taskkill /T` path is exercised by the existing Windows CI browser job;
this local run verified the Unix process-group path.
