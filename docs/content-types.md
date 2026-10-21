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

**Batch (admin API).** `POST /api/v1/admin/content/_batch` applies up to 200 `create`, `update` and `delete` operations in one transaction: all of them are saved or none. Body: `{"operations":[{"op":"create","type":"companies","fields":{"name":"Acme"}},{"op":"create","type":"contacts","fields":{"full_name":"Al","company":"$ref:0"}}]}`. `"$ref:N"` stands for the id of the record created by operation N, so related records can be created together. `update` needs `id` and the record's current `version`. On success the response lists `results` (index, type, id, record). On failure nothing is saved and the error names the operation: field errors come back as `operations[1].full_name`. It needs the same write permission as a single save.

Tools: `hydro content-type new <id>` writes a skeleton; `hydro content-type validate` loads every file and reports errors. Restart `hydro serve` after changing a file.

Stock types that ship with the repository (`content-types/`) are copied into your project by `hydro new --preset ...`; delete the ones you do not need.
