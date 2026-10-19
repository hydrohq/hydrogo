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

Public API: `GET /api/v1/content/<id>?limit=&offset=&sort=-field&search=&filter[field]=&locale=` and `GET /api/v1/content/<id>/<record-id>`. Responses use the standard `{data, metadata}` envelope.

Tools: `hydro content-type new <id>` writes a skeleton; `hydro content-type validate` loads every file and reports errors. Restart `hydro serve` after changing a file.

Stock types that ship with the repository (`content-types/`) are copied into your project by `hydro new --preset ...`; delete the ones you do not need.
