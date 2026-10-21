# Content types

Anything that is not pages, news or members is a content type: one YAML file in `content-types/`. HydroGo derives storage and validation, the admin screens (`/admin/content/<id>`) and, with `public: true`, a read API (`/api/v1/content/<id>`). No Go, no Svelte.

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/ozyildirims/hydrogo/main/docs/schemas/content-type.schema.json
id: recipes
label: Recipes
title_field: title
public: true      # serve published records at /api/v1/content/recipes
drafts: true      # records start as drafts; the public API returns only published ones
revisions: true   # keep the last 50 versions, with compare and restore
fields:
  - {id: title, label: Title, kind: text, required: true, list: true, searchable: true}
  - {id: course, label: Course, kind: select, options: [starter, main, dessert], list: true, filterable: true}
  - {id: photo, label: Photo, kind: media}
  - {id: author, label: Author, kind: relation, to: team}
  - {id: method, label: Method, kind: richtext}
```

Field kinds: `text`, `textarea`, `richtext`, `number`, `boolean`, `date`, `datetime`, `select`, `media`, `relation` (`to:` the target type, `many: true` for several). Field flags: `required`, `unique`, `list`, `searchable`, `filterable`, `localized`, `min`, `max`.

The complete reference is the header comment of [`internal/contenttypes/definition.go`](../internal/contenttypes/definition.go); the machine-readable form is [`schemas/content-type.schema.json`](schemas/content-type.schema.json) (add the `yaml-language-server` line above for completion and validation in VS Code).

Public API: `GET /api/v1/content/<id>?limit=&offset=&sort=-field&search=&filter[field]=&locale=` and `GET /api/v1/content/<id>/<record-id>`. Responses use the standard `{data, metadata}` envelope. A type holds up to 100,000 records per tenant by default; change it with `limits.records_per_type` in `config.yaml` (a negative value removes the cap). `offset` is limited to 10,000. To walk a long list, pass `metadata.next_cursor` from a page as `?after=` on the next request (no offset; it works with the default order and `sort=created_at`, `-created_at`, `updated_at`, `-updated_at`).

Filters work on fields marked `filterable` (text, textarea, select, number, boolean, date, datetime and single relation). `filter[field]=v` is an exact match; `filter[field][op]=v` takes an operator: `ne` (also matches records without a value), `gt`, `gte`, `lt`, `lte` (number and date fields; a date such as `2026-10-01` means midnight UTC), `in` (comma separated, at most 50 values) and `null` (`true` for records without a value, `false` for those with one). On PostgreSQL every `filterable` field is indexed when the server starts (building the first indexes on a large table takes a few seconds), which keeps filtered and cursor-paged lists fast on hundreds of thousands of records. Combine two operators on one field for a range: `filter[amount][gte]=100&filter[amount][lt]=900`. `expand=company,partners` adds the records that those relation fields point to, next to the plain ids, under `expanded` (an object per single relation, an array for `many: true`), so a list can show a company name without one request per row. It works on lists and single records, accepts up to five fields, and on the public API only expands relations to `public: true` types and leaves out related drafts. For a relation field the value is the related record's id, so `filter[company]=<id>` lists the records that belong to one company.

**Sequence numbers.** A text field with `sequence: {prefix: "INV-{yyyy}-", pad: 5}` is numbered by the server: `INV-2026-00001`, `INV-2026-00002`, and so on, without gaps. `{yyyy}`, `{yy}` and `{mm}` in the prefix are the current UTC year and month, and the counter restarts whenever the expanded prefix changes (a new year starts again at 1). The number is taken in the same transaction as the record, so a rejected or rolled-back save does not use one up. It is assigned on create only, never changes, and a value sent by a client is ignored; the field is unique and read-only in the admin. Deleting a record leaves its number unused, as invoicing rules usually require (void instead of delete).

**Money and computed fields.** `decimals: 2` on a number field fixes its decimal places: `12.345` is rejected instead of rounded, and so is a value too large to be held exactly (more than 15 digits including the places). `computed: "quantity * unit_price"` makes the server calculate a number field on every save with exact decimal arithmetic (no floating point), rounding half away from zero to the field's `decimals`. It reads number fields declared before it, so totals can build on subtotals: `subtotal` = `quantity * unit_price`, `tax` = `subtotal * 0.18`, `total` = `subtotal + tax`. Operators are `+ - * /` and parentheses; functions are `abs(x)`, `min(a, b)`, `max(a, b)` and `round(x, places)`. An empty field counts as 0; dividing by zero, or a result that does not fit, rejects the save. The value is stored and returned like any field (so it can be filtered and sorted), a value sent by a client is ignored, and the field is read-only in the admin.

**Rules and states.** Type-level `rules` check and lock records, and `states` turns a select field into a small state machine:

```yaml
rules:
  - assert: "quantity > 0"                  # every saved record must satisfy it
    message: "Quantity must be positive"
  - when: "status == 'posted'"              # a stored record matching `when` is locked:
    readonly: true                          # no edits, no delete
    message: "Posted invoices are locked"
states:
  field: status
  initial: draft
  transitions:
    - {from: draft, to: approved, permission: invoice.approve}
    - {from: approved, to: posted, when: "total > 0", message: "Nothing to post"}
    - {from: posted, to: void}
```

Conditions use the same expression language as computed fields plus comparisons (`== != < <= > >=`), `&& || !`, `'text'`, `true`, `false` and `empty(field)`, and may only read fields of the type (an unset field equals `""`, `0` and `false`). An `assert` with `when` only applies while `when` holds for the saved record. A failed rule answers 422 with the rule's message (and `fields._rule`). The state field is read-only: new records start in `initial` whatever the client sends, a normal save keeps the stored state, and a change goes through `POST /api/v1/admin/content/<type>/<id>/state` with `{"to": "approved", "version": 3}`. A step that does not exist, or whose `when` fails, answers 422; a step with a `permission` needs a user who holds it (admins hold every permission) and answers 403 otherwise. Readonly rules do not block transitions, so a posted invoice can still be voided, and asserts run on the record as it will be after the change.

**Batch (admin API).** `POST /api/v1/admin/content/_batch` applies up to 200 `create`, `update` and `delete` operations in one transaction: all of them are saved or none. Body: `{"operations":[{"op":"create","type":"companies","fields":{"name":"Acme"}},{"op":"create","type":"contacts","fields":{"full_name":"Al","company":"$ref:0"}}]}`. `"$ref:N"` stands for the id of the record created by operation N, so related records can be created together. `update` needs `id` and the record's current `version`. On success the response lists `results` (index, type, id, record). On failure nothing is saved and the error names the operation: field errors come back as `operations[1].full_name`. It needs the same write permission as a single save.

Tools: `hydro content-type new <id>` writes a skeleton; `hydro content-type validate` loads every file and reports errors. Restart `hydro serve` after changing a file.

Stock types that ship with the repository (`content-types/`) are copied into your project by `hydro new --preset ...`; delete the ones you do not need.

## Per-type permissions

By default a type is readable with `admin.content.read` and writable with
`admin.content.write`. To restrict one, name its own permissions:

```yaml
permissions: {read: deals.read, write: deals.write}
```

Either side may be left out (that direction keeps the generic permission).
When a side is named, only that permission counts: editors do not get it by
role. Admins hold every permission; for anyone else grant it per user on the
Users screen. Names look like `area.action` and may not start with `admin.`.
Writes cover create, update, delete, state steps and `_batch`; a batch fails
as a whole with 403 if any operation touches a type the caller cannot write.
The `permission` of a state step is added to the same catalog.

## Row access

```yaml
access: {owner: owner, all: deals.read_all}
fields:
  - {id: owner, label: Owner, kind: text, filterable: true}
```

In the admin API, a user without `deals.read_all` only sees records whose
`owner` equals their e-mail address (compared exactly; accounts keep it in
lower case, so write owners in lower case). Lists are filtered in the database (so
paging and totals stay right), reading, editing, deleting, revisions, state
steps and `_batch` on someone else's record answer 404, `expand` leaves out
related records they do not own, and whatever they create or save gets them
as the owner (a value sent for `owner` is replaced). Admins and holders of
the permission see everything. The permission joins the catalog (grant it per
user). The public API (`/api/v1/content/...`) has no users and is not
affected, so a type with `access` cannot be `public`.

## Field permissions

```yaml
- {id: margin, label: Margin, kind: number, decimals: 2, permission: deals.margin}
```

Users without the permission (admins always have it) do not get the field in
admin reads, anything they send for it is ignored and the stored value stays
when they save, and filtering, sorting, grouping or summing by it answers 400.
The field cannot be `required`, `unique`, `searchable`, `localized` or a
sequence, and a type with such fields cannot be `public`. The admin forms
still list the field.

## Aggregation

```
GET /api/v1/admin/content/deals/_aggregate?group_by=stage&sum=amount&filter[owner]=ann@x.test
```

returns `{data: {total: {count, sums}, groups: [{key, count, sums}]}}`.
`group_by` takes a text, select, boolean, date or single relation field,
`sum` up to ten number fields; sums are exact decimals (strings, with the
field's `decimals`). The list filters, `search` and `status` apply, as do row
access and field permissions. The server totals the matching records itself,
so a request that matches more than 20,000 records or forms more than 1,000
groups is answered 422: narrow it with filters.

## CSV export and import

`GET /api/v1/admin/content/<type>/_export` streams the records matching the
list filters as CSV (`id`, then every field the caller may read; lists are
joined with `;`). Cells that start with `=`, `+`, `-`, `@` or a tab are
prefixed with `'` so a spreadsheet does not run them. Row access and field
permissions apply; more than 50,000 matching records end the file with a
`# export incomplete` row (narrow the filters).

`POST .../_import` takes `Content-Type: text/csv`: the first row names fields
(computed, sequence and state fields and `id` are refused), every further row
creates a record. Numbers, booleans (`true/false/1/0/yes/no`) and `;`-separated
relation lists are converted. The whole file is one transaction: the first
bad row answers 422 with `rows[i].<field>` codes and nothing is saved. Add
`?dry_run=1` to validate without saving. Limits: 1,000 rows and 5 MB;
write access, row access and rules apply as for single saves.

## Service accounts

A side service (accounting, an importer, a nightly job) should not borrow a
person's login. Create it its own identity:

```bash
hydro service-account create --name ledger --permissions invoices.read,invoices.write --expires-days 90 --config config.yaml
```

It prints a bearer token once. The account is a viewer that nobody can sign
in as (random, unseen password) with exactly the permissions you listed on
top; the token expires (default 90 days, at most 365) and is sent as
`Authorization: Bearer <token>` to `/api/v1/admin/...`. Audit entries and
revisions name it `service+ledger@hydrogo.invalid`. Permissions that no
content type declares (type `permissions`, `access.all`, field `permission`,
state-step `permission`) are ignored by the server, so a typo fails closed.
Note that, as a viewer, it can still read types that use the generic content
permissions. Rotate by creating a new account name; end one with
`hydro service-account revoke --name ledger` (tokens revoked, account
disabled).

## Hooks: ask a side service first

Domain logic (posting to a ledger, a credit check, e-invoicing) lives in a
side service, not in HydroGo. A type can make a change wait for it:

```yaml
hooks:
  - {on: transition, to: posted, service: ledger}    # state steps (from/to optional)
  - {on: create, service: credit-check, fail_open: true}
  - {on: delete, service: ledger}                    # also: update
```

The services are defined by the operator, never by a content type, so a
content-type author cannot make the server call arbitrary URLs. In
`config.yaml`:

```yaml
hooks:
  - id: ledger
    url: http://127.0.0.1:9100/hydro      # may be a private address
    secret_env: LEDGER_HOOK_SECRET        # at least 16 characters
    timeout: 5s                           # 100ms to 10s
```

HydroGo refuses to start when a type names a service that is not defined.
Before the change is saved it sends one POST with the webhook signature
headers (`X-Hydro-Signature`, `X-Hydro-Timestamp`, `Idempotency-Key`,
`X-Hydro-Event: hook.<type>.<on>`) and the body
`{"data": {"type", "on", "id", "from", "to", "actor", "fields"}}`.
The answer decides: 2xx saves; 4xx rejects the change with 422 and the
service's `{"message": "..."}`; a timeout, a redirect, a 5xx or an
unreachable service rejects it with 503 (fail closed), except for
`fail_open: true`, which lets the change through. In safe mode no hook is
reachable. Hooks run before the change and outside its transaction, so the
service judges what was submitted; versions still guard the save. In a
`_batch` all hooks run first, before the transaction (with `$ref:N` values
unresolved). Keep the service fast and idempotent: a change may be retried.

### Schedules

A hook service can also be called on a clock, to run nightly checks or
close periods. Add `schedules` to the service in `config.yaml`:

```yaml
hooks:
  - id: ledger
    url: http://127.0.0.1:9100/hydro
    secret_env: LEDGER_HOOK_SECRET
    schedules:
      - {name: overdue-check, cron: "0 2 * * *"}   # every day at 02:00 UTC
```

At each matching minute (five-field cron, UTC) HydroGo sends the service a
signed request with `X-Hydro-Event: hook.schedule.<name>` and
`{"data": {"type": "schedule", "on": "<name>", "fields": {"at": "<minute>"}}}`.
The service then does its work, typically by calling the admin API with its
service-account token. A call that is still running is not started again,
minutes missed while the server was down are not caught up, and every server
instance fires its own calls, so run one instance per schedule (or make the
service idempotent per `at`). Failures are logged as warnings.

## The erp-lite preset

`hydro new --preset erp-lite` starts a back office for small trade
businesses: `products`, `warehouses`, `accounts` (customers and suppliers),
`sales_orders` with `order_lines`, `invoices` with `invoice_lines`, and
`payments`. It is a starting point to edit, not an accounting system:

- Documents are numbered without gaps (`SO-2026-00001`, `INV-…`, `PAY-…`).
- Money has two decimals, line totals and document totals are computed on
  the server (`quantity * unit_price`, `subtotal - discount + tax`).
- Orders and invoices have state machines. Posting, voiding, fulfilling and
  cancelling need `erp.invoices.post`, `erp.invoices.void`, `erp.orders.fulfil`
  and `erp.orders.cancel`; admins hold them, give them to others per user (or
  to a service account).
- A posted, paid or void invoice and a closed order cannot be edited.
- Create a document and its lines in one `_batch` request (`"order":
  "$ref:0"`), so a failed line saves nothing; the totals follow the lines.

Order and invoice subtotals and taxes are totalled from their lines by the
server (`rollup`), and the lines of a posted invoice are locked with it.
Books are closed by adding a `period_closings` record (needs
`erp.periods.close`): orders, invoices and payments dated on or before the
latest closing date can no longer be created, changed, moved on a state step
or deleted, and closings themselves cannot be edited or removed.

What it deliberately leaves to you or to a side service: there is no stock
ledger or accounting journal. Add those as a service that is called
through `hooks` (for example `on: transition, to: posted`) and works through
a service account.

## Rollups: totals from related records

```yaml
# invoices.yaml
- {id: subtotal, label: Subtotal, kind: number, decimals: 2, rollup: {from: invoice_lines, via: invoice, sum: line_total}}
- {id: total, label: Total, kind: number, decimals: 2, computed: "subtotal + tax"}
```

`from` is the type holding the lines, `via` its single relation field that
points back at this type, `sum` the number field to add up (leave it out to
count). The field is read-only: clients cannot set it. The server recalculates
it, and the computed fields that read it, in the same transaction as every
create, update, delete or move of a line to another parent, so it is never
stale. Declare rollups before the computed fields that use them.

A line cannot be changed while its parent is locked by a readonly rule (a
posted invoice): that would change the parent's total, so the change is
refused like any edit of the locked parent. every line write also rewrites its parent (its version goes up). On
PostgreSQL concurrent writes to the lines of one parent queue on the parent
row; MySQL's default isolation may leave a total briefly stale if two lines of
one parent are written at the same moment, and the next line change corrects
it (MySQL is not exercised by the test suite here).

## Period lock

```yaml
period_lock: {field: issue_date, closings: period_closings, through: closed_through}
```

Records whose `field` (a date) is on or before the latest `through` date in
the `closings` type can no longer be created, changed, moved on a state step
(including voiding) or deleted, and a date cannot be edited into a closed
period. Closing a period is adding a record to the closings type, so who may
close is that type's `permissions`; make it append-only with
`rules: [{when: "true", readonly: true}]` (see `content-types/erp/`). Startup
fails if `field` is not a date field of the type or `closings`/`through`
do not exist. Totals the server rewrites on a parent because a line changed
(rollups) are not checked against the period: lock the parent with a readonly
rule too.
