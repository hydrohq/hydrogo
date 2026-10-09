# Admin izin ve rol denetimi

**Tarih:** 2026-10-08  
**Sonuç:** Auth rol × izin × Admin route matrisi ve mutation ret testleri geçti  
**Kapsam:** Edition rol varsayılanları, kayıtlı izinler, kullanıcı yönetimi
mutasyonları ve yöneticinin kendi erişimini değiştirme girişimi.

Doğrulama komutu:

```sh
TMPDIR=/var/tmp/hydro-go-tests go test ./internal/auth ./cmd/hydro -count=1
```

Testler şu davranışları doğrular:

- `/admin/control`, security, operations, users, permissions, site, developer,
  resource usage ve account GET route'ları aktif/inaktif Admin, Editor, Viewer
  için; bilinen her izin override'ının varsayılan/allow/deny etkisiyle çapraz
  test edilir. Site, operations ve users-management route'larının role ek
  koşulları da kontrol edilir.
- Admin/editor/viewer varsayılan izinleri ve etkin kullanıcı izin override'ları.
- Edition kataloğunda olmayan izinlerin yetki vermemesi ve izin yazımından
  filtrelenmesi.
- Editor ve viewer rollerinin rol, izin ve hesap kapatma mutasyonlarından
  reddedilmesi.
- `admin.users.manage` izni açıkça reddedilmiş yöneticinin kullanıcı yönetimi
  mutasyonlarından reddedilmesi.
- Yöneticinin kendi rolünü veya kendi izinlerini değiştirme isteğinin `409`
  olması ve izin deposuna yazma yapılmaması.
- Tek Edition adminiyle rol ve permission form akışının gerçek Chromium
  testinde çalışması; editör deny/allow override ve rol değişiklikleri.
- Editor ve viewer oturumlarının geçerli CSRF tokenıyla başka kullanıcı için
  rol değiştirme, izin override'ı yazma ve hesabı kapatma POST'larından `403`
  alması.
- Editor ve viewer'ın rol, permission, disable, invitation ve password-reset
  user-management mutation endpoint'lerinden `403` alması.
- Editor ve viewer'ın geçerli CSRF tokenıyla IP ban ekleme/kaldırma ve backup
  POST'larından `403` alması; geçerli CSRF admin/user management akışlarının
  kendi izin kurallarına göre çalışması.

İlgili Chromium doğrulama komutu:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project chromium --workers=1 \
  --grep 'editor and viewer sessions' --reporter=line
```

Bu, tek tek davranış kanıtıdır. Her route × rol × CSRF durumu × override
değerini kapsayan çapraz matris Go route-policy testinde üretilmiştir. Browser
testi CSRF tokenlı mutation ve CSRF'siz form retlerini ayrıca doğrular.
PostgreSQL auth sözleşme testi önceki doğrulama aşamasında geçmiştir. Bu aşama
çalışma zamanı veya veritabanı davranışını değiştirmez.

Generated Admin resource policy contract also verifies that a collection
denied by `CanReadResource` is omitted from the resource index and that its
list/create/record routes return `404` without calling the API. A valid-CSRF
generated create whose resource API denies the write returns a user-facing
`403` and never redirects as if it succeeded.

`TestGeneratedAdminMutationUsesCSRFAndResourceAPI` sends requests directly
through the registered generated handler in an in-memory mux. It checks that a
missing CSRF token is rejected and a valid token reaches the Edition resource
API, without requiring a loopback listener:

```sh
go test ./internal/adminstudio -run 'TestGeneratedAdminMutationUsesCSRFAndResourceAPI|TestGeneratedAdmin(ResourceReadPolicyHidesIndexAndBlocksCollectionRoutes|ResourceCapabilitiesHideAndDenyBasicCRUD|ResourceCapabilitiesAllowUpdateWithoutCreateOrDelete|GlobalRequiresUpdateCapability)' -count=1
```

ADR-0361 adds optional Edition-provided `ResourceAccess` capabilities for
basic generated CRUD. Handler tests prove that denied create/update/delete
actions disappear, direct routes return `403` before a mutation reaches the
resource API, and update can be allowed independently from create/delete.
Draft workflow and other editorial mutations keep their dedicated callbacks.
The generated block Chromium journey also verifies read-only list state,
valid-CSRF create/update/delete denials, and unchanged persisted data; the full
generated editorial suite passed 7/7 on SQLite and PostgreSQL 18. Database
behavior and migrations are unchanged.

ADR-0362 verifies that a readable record without update capability opens a
read-only detail page using escaped field values and no forms. POST update and
delete remain denied before the API is called. Draft and global-resource
behavior keep their dedicated policy rules.

## Dynamic Collections mutation CSRF and role matrix — 2026-10-08

The test builds the actual registered Collections router and wraps it with the
production double-submit CSRF middleware. All nine registered admin/API
mutation routes (schema create/delete, record create/update/delete) are checked
three ways: an Admin request with a mismatched token is rejected with 403;
Editor and Viewer requests with a valid token are denied by the route's Admin
role check with 403. The handler uses an empty Service, so these results also
prove denial occurs before any repository operation.

```sh
go test ./internal/collections/... -count=1
go vet ./internal/collections/...
```

Both commands passed locally. This matrix covers denied mutations; successful
CRUD persistence is covered by the existing Dynamic Collections SQLite
Chromium journey. It does not claim a PostgreSQL browser result.

## Media Admin explicit permission-deny matrix — 2026-10-09

The real registered media Admin routes now include an active Admin with an
explicit `media.manage=false` override in addition to an Editor with the same
deny and a default Viewer. List, upload, metadata, folder, settings, and
visibility routes return 403 for each user. For mutation routes, a mismatched
CSRF token is rejected by the production middleware, and a valid CSRF token
then reaches the media permission boundary and is denied before handler
dependencies. Media cleanup is intentionally kept as a separate Admin-only
capability and is not included in the `media.manage` override assertion.

```sh
go test ./internal/media -run 'TestMediaAccessUsesPermissionOverrides|TestDeniedMediaPermissionStopsAdminHandlersBeforeDependencies|TestAdminMediaSettingsPersistAndApplyHotlinkProtection' -count=1
go vet ./internal/media
```

Both passed. Existing successful Editor settings persistence and visibility
tests remain in the same package. This adds the explicit Admin-deny cells for
media management; it does not close the repository-wide route × role matrix.

## Media Admin explicit-deny persisted state — 2026-10-09

The SQLite registered-route contract now executes an active Admin with
`media.manage=false` against metadata, visibility, upload, and settings
mutations using valid CSRF. Every request returns 403. SQL readback confirms
metadata, visibility, and tenant settings stay unchanged; upload leaves both
the asset table and upload directory unchanged. Existing valid-CSRF Editor
mutations then succeed and persist, proving the denied requests did not poison
the success path. The fixture resolves tenant scope from the host and confirms
cross-tenant access remains isolated.

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./tests/contract -run '^TestSQLiteAdminMediaMutationRoutesCSRFAndPersistence$' -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./tests/contract -run '^Test(SQLite|Postgres)AdminMediaMutationRoutesCSRFAndPersistence$' -count=1 -p 1 -v
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go vet ./tests/contract
```

SQLite passed. The PostgreSQL case was skipped because
`HYDRO_TEST_POSTGRES_URL` is unset; it is not a PostgreSQL pass. The full
contract package was not rerun as part of this focused change.

## Admin push broadcast route — 2026-10-08

The successful broadcast test now mounts the real `Handler.Routes()` registry
behind the shared router authorization and production CSRF middleware. An
active Admin with the matching cookie/header token sends to the selected
mobile audience; the redirect summary and audit event are checked. A deny
matrix confirms that a bad token for Admin and valid tokens for Editor and
Viewer all return 403 without dispatching or recording an audit event.

```sh
go test ./internal/push/... -count=1
go vet ./internal/push/...
```

Both commands passed. The Firebase sender unit test uses an in-memory HTTP
RoundTripper, so this is not a delivery test against FCM or a browser push
permission flow.

## Trash and restore route permission matrix — 2026-10-08

Delete and restore requests now go through the registered trash routes, an
active-session router authorizer, and production CSRF middleware. Wrong tokens
for both content and submission routes return 403 without calling the service.
With valid tokens, the matrix verifies Editor's default page permission and
submission denial, Viewer denial, explicit content-write denial for Editor and
Admin, explicit site-management denial for Admin, and an explicit site grant
allowing an Editor to restore a submission.

```sh
go test ./internal/trash/... -count=1
go vet ./internal/trash/...
```

Both passed. These checks cover the handler/authorization boundary; they do not
certify trash repository persistence or the browser restore journey.

## Mailer Admin mutation routes — 2026-10-08

Tests wrap the app's Admin branding-save and test-email/SMTP-ping URL patterns
with the active-session router and production double-submit CSRF middleware.
Valid Admin branding writes persist to the settings fake; valid Admin test
email and SMTP ping reach their fake sender. Invalid Admin CSRF and valid
Editor/Viewer requests receive 403 and produce no write/send side effect.

```sh
go test ./internal/mailer -run 'TestEmailBrandingHandlerRequiresAdminAndPersistsValidatedFields|TestEmailBrandingMutationRouteRejectsCSRFAndNonAdminRoles|TestTestPreview(SendsTextAndHTMLToAdmin|RejectsEditor|SMTPPingDoesNotRequireVerifiedEmail|RequiresVerifiedAdminAddress)' -count=1
go vet ./internal/mailer/...
```

The focused tests and vet passed. A full `go test ./internal/mailer/... -count=1`
attempt was blocked by `TestSMTPPingChecksGreetingWithoutSendingEmail` and
`TestSMTPSendAppliesTimeoutBeforeGreeting` failing to bind loopback TCP in this
environment. No real SMTP service or browser result is claimed.

## Webhook test delivery route — 2026-10-08

The test sandbox mutations are exercised through the app's exact route
patterns, an active-session router, and production CSRF middleware. A valid
Admin request dispatches one signed `page.published` test event and verifies
the timestamp-based HMAC, event header, initiator, and receiver response.
Admin with a bad CSRF token and Editor/Viewer with valid tokens all receive 403
without dispatching. Dispatcher retry/signature/backoff/redirect tests and the
worker dead-letter test use deterministic RoundTripper responses, avoiding
loopback network dependencies while retaining HTTP request contracts.

```sh
go test ./internal/webhooks/... -count=1
go vet ./internal/webhooks/...
```

Both commands passed. The fake receiver proves the local dispatcher contract,
not delivery to an external webhook endpoint or a browser journey.

## Generated CRUD CSRF matrix — 2026-10-09

Generated Admin basic mutations now check missing/invalid CSRF across create,
update, and delete. Each request receives 403 before the Edition resource API
is called; a valid-CSRF create reaches the API once and persists the submitted
fields through the adapter contract.

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/adminstudio -run '^TestGeneratedAdminMutationUsesCSRFAndResourceAPI$' -count=1
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/adminstudio -count=1
```

Both passed. This closes the basic generated CRUD CSRF cell; it does not close
the complete generated editorial, media, locale, revision, and draft mutation
matrix or the browser route matrix.

## Generated editorial CSRF denials — 2026-10-09

The generated revision restore, draft workflow transition, and draft discard
handlers now have explicit missing-CSRF cases. Tests assert `403` before
revision reads or Edition persistence callbacks, then retain the valid-CSRF
restore/transition/discard checks. The draft save missing-CSRF case remains in
the same test.

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/adminstudio -run 'TestGeneratedAdminRevisionHistoryDiffAndRestoreAreBounded|TestGeneratedAdminDraftsStayIsolatedAndUseTwoVersionCAS' -count=1
```

Passed. Localization copy/save and the full browser role × route matrix remain
open.

The same test now also checks that localized save and locale copy without
CSRF return `403` before their Edition storage callbacks. Valid-CSRF translation
save and copy still pass in the same locale-scoped fixture:

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/adminstudio -run '^TestGeneratedAdminLocalizesOnlyOptedInFieldsThroughEditionStore$' -count=1
```

Passed. Browser locale navigation and cross-role route behavior remain separate
acceptance cells.

## Built-in Admin and media API mutation route CSRF matrix — 2026-10-09

The test derives unsafe Admin and media API routes from the real Site, News,
Pages, Media, and Auth `Routes()` outputs. Site is checked in core and Forms +
Fonts compositions; News and Pages are checked with their editor routes; Media
includes optional unused-asset cleanup and its API upload/folder/metadata
mutations; Auth includes identity, user, and account administration. It
verifies the composition maps each route to `router.Protected`, then sends each
exact method/path through the production CSRF middleware without a token and
requires `403`. Public form submission, reset-token, and public media delivery
routes are excluded.

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./cmd/hydro -run '^TestBuiltInMutationRoutesAreProtectedAndCSRFChecked$' -count=1
```

All six route compositions passed. Persistence outcomes and role override
behavior for every Site/News/Pages/Media/Auth action remain covered by
handler-specific tests or open cells in the full acceptance matrix.

Validation for the read-only detail change:

```sh
TMPDIR=/var/tmp go test ./internal/adminstudio ./extension/v1/adminstudio ./examples/project-edition/custom -count=1
cd e2e && npx playwright test custom/generated-richtext.spec.ts --project=chromium --workers=1 --reporter=line --output=/tmp/hydro-generated-readonly-full
```

```sh
TMPDIR=/var/tmp go test ./internal/adminstudio -run 'TestGeneratedAdmin(ResourceReadPolicy|SurfacesResourceAPIDenial|ResourceCapabilities)' -count=1
TMPDIR=/var/tmp go test ./internal/adminstudio -count=1
go test ./extension/v1/adminstudio ./examples/project-edition/custom -count=1
cd tests/external-edition && go test ./... -count=1
```

### Site settings CSRF × permission × persisted state — 2026-10-09

The registered `POST /admin/site/settings` handler is now exercised behind the
production CSRF middleware with SQLite migrations and the SQL store. Invalid
CSRF and an Admin `site.manage=false` override both return 403 and leave the
setting absent. A valid-CSRF Editor with an explicit `site.manage=true` grant
receives 204 and the setting is read back from SQLite. The focused test, full
`modules/site/transport` package and vet passed. This closes one Site settings
mutation cell only; it does not imply the full Site/Admin route matrix passed.

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 go test ./modules/site/transport -run '^TestAdminSiteSettingRouteEnforcesCSRFBeforePermission$' -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 go vet ./modules/site/transport
```

### News content permission × CSRF routes — 2026-10-09

The News content-read/write route permission table now runs behind the real
CSRF middleware. All 13 mutation routes reject a wrong token with 403; all 15
read/write route cases then use a valid token to check Editor, Viewer grants,
Admin default/deny, and inactive grant behavior against content permissions.
The News transport package tests and vet passed. This is route authorization
evidence only: configured storage, successful writes, and persisted state remain
covered by separate Edition/browser/contract tests.

### Pages content permission × CSRF routes — 2026-10-09

The Pages content-write route test now also composes the real CSRF middleware.
All four mutation routes reject a wrong token with 403; the read/create/update/
publish route cases then check valid-token Editor, Viewer grant, Admin default/
deny, and inactive grant behavior. The full Pages transport package and vet
passed. This verifies authorization routing, not successful DB mutation or
persisted state. News and Pages follow the same test boundary.

### User-management mutations × CSRF — 2026-10-09

Admin role change, permission change, account disable, invitation issue, and
admin password-reset-link handlers now run behind the real CSRF middleware in
their permission-denial tests. Every handler returns 403 for invalid CSRF before
authorization; with valid CSRF the Admin `users.manage=false` override and
Editor/Viewer roles still return 403. The full `internal/auth` package and vet
passed. This proves denied handler access only; successful persistence and the
complete live-session browser flow remain separate evidence.

### Site Admin mutation route × CSRF × `site.manage` — 2026-10-09

Every Admin POST route registered by the Site fixture with Forms and Fonts
enabled is now exercised through `http.ServeMux` and the real CSRF middleware.
Invalid tokens return 403; valid-token Editor/Viewer defaults, Admin explicit
deny, and inactive explicit grant also return 403 before handler dependencies.
This caught `/admin/site/link-check` returning 503 for denied users when its
scanner dependency was absent. Authorization now returns 403 before the
separate authorized-but-unavailable 503 path. The full Site transport package
and vet passed. This covers registered Forms/Fonts routes; the optional AI
summary and mail-spool retry routes are covered separately below. Queue
rendering and successful DB-backed mutation state beyond Site settings remain
open.

### Optional Site AI summary mutation — 2026-10-09

With Forms enabled and an explicit AI provider/audit writer, the registered
`POST /admin/site/inbox/{form}/ai-summarize` route rejects invalid CSRF for
Admin and rejects valid-CSRF Editor, Viewer, Admin `site.manage=false`, and
inactive Admin with an explicit grant. The checks run through the production
CSRF middleware and registered `http.ServeMux` route; denial happens before
service, storage, audit, or provider work. The Site transport package tests and
vet passed. This does not verify a successful AI summary or mail-spool queue
rendering/delivery.

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 \
  go test ./modules/site/transport -run '^TestOptionalSiteAssistantMutationEnforcesCSRFAndSiteManage$' -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 \
  go test ./modules/site/transport -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 \
  go vet ./modules/site/transport
```

### Site mail-spool retry success and denial — 2026-10-09

The registered `POST /admin/site/mail/{id}/retry` route now has an integrated
CSRF and `site.manage` test with a recording spool. Invalid CSRF, Editor,
Viewer, and Admin `site.manage=false` all return 403 without retrying. An
authorized Admin with valid CSRF receives the expected redirect and retries
exactly the requested message once. The Site transport package tests and vet
passed. Queue rendering and delivery to a real mail service remain unverified.

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 \
  go test ./modules/site/transport -run '^TestSiteMailRetryRequiresPermissionAndPersistsOnlyAuthorizedAction$' -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 \
  go test ./modules/site/transport -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 \
  go vet ./modules/site/transport
```
