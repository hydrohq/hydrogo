# Page and News denied mutations storage isolation — 2026-10-10

## Verified locally

Added spy repositories and dedicated tests in both `modules/pages/transport/permissions_test.go` and `modules/news/transport/permissions_test.go`:

```sh
go test ./modules/pages/... ./modules/news/... -count=1
go vet ./modules/pages/... ./modules/news/...
```

Both package test suites passed.

## Contract coverage

The test verifies that when mutations are denied by authorization policies or CSRF validation:
- **Pages**: Viewer role, Editor role with `PermissionContentWrite: false`, and Admin role with missing or invalid CSRF attempting `POST /admin/pages`, `POST /admin/pages/{id}`, `POST /admin/pages/{id}/publish` are rejected with `403 StatusForbidden`. The spy repository asserts zero invocations of `CreateDraftContent`, `UpdateDraftContent`, or `Publish`.
- **News**: Viewer role, Editor role with `PermissionContentWrite: false`, and Admin role with missing or invalid CSRF attempting `POST /admin/news`, `POST /admin/news/{id}`, `POST /admin/news/{id}/publish`, `POST /admin/news/{id}/retract`, `POST /admin/news/{id}/schedule` are rejected with `403 StatusForbidden`. The spy repository asserts zero invocations of `CreateDraft`, `UpdateDraft`, `Schedule`, `PublishNow`, or `Retract`.

This proves that denied editorial mutations fail closed at the transport/security boundary before reaching persistence dependencies.
