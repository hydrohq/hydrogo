import { type DocSection, page, p, h2, h3, code, tabs, note, tip, warn, table, ul, ol } from './types';

export const contentTypes: DocSection = {
  id: 'content-types',
  title: 'Content types',
  pages: [
    page(
      'overview',
      'Content types',
      "Model anything beyond pages and news with one YAML file. HydroGo derives storage, validation, admin screens and a public API from it.",
      p("Anything that is not a page, a news article or a member is a **content type**: one YAML file in `content-types/`. No Go and no Svelte are involved. From the file HydroGo derives:"),
      ul(
        "database storage and validation,",
        "admin screens at `/admin/content/<id>` (list, search, filters, edit form, drafts, revisions),",
        "with `public: true`, a read API at `/api/v1/content/<id>` (see [Content API](doc:rest-api/content-api)).",
      ),
      h2('A complete example'),
      code('yaml', `# yaml-language-server: $schema=https://raw.githubusercontent.com/ardaozyildirim/hydrogo/main/docs/schemas/content-type.schema.json
id: recipes
label: Recipes
description: Dishes shown on the food blog.
title_field: title
public: true      # serve published records at /api/v1/content/recipes
drafts: true      # records start as drafts; the public API returns only published ones
revisions: true   # keep the last 50 versions, with compare and restore
fields:
  - {id: title,  label: Title,  kind: text,     required: true, list: true, searchable: true}
  - {id: course, label: Course, kind: select,   options: [starter, main, dessert], list: true, filterable: true}
  - {id: photo,  label: Photo,  kind: media}
  - {id: author, label: Author, kind: relation, to: team}
  - {id: method, label: Method, kind: richtext, searchable: true}`, 'content-types/recipes.yaml'),
      p("Restart `hydro serve` after adding or changing a file. Unknown YAML keys are rejected, so a typo fails loudly instead of being ignored."),
      h2('Type keys'),
      table(
        ['Key', 'Required', 'Meaning'],
        ['`id`', 'yes', 'Lowercase letters, digits and underscores, starting with a letter (max 63 characters). Becomes the URL segment in `/api/v1/content/<id>` and `/admin/content/<id>`.'],
        ['`label`', 'yes', 'Human name shown in the admin.'],
        ['`fields`', 'yes', 'At least one field. See [Fields](doc:content-types/fields).'],
        ['`description`', 'no', 'Help text under the title in the admin.'],
        ['`group`', 'no', 'Admin navigation group that contains this type.'],
        ['`icon`', 'no', 'Icon for the navigation entry.'],
        ['`title_field`', 'no', 'Field whose value is shown as the record title.'],
        ['`public`', 'no', 'Serve published records at `/api/v1/content/<id>`. Types that are not public answer 404, exactly like unknown types.'],
        ['`drafts`', 'no', 'Add a draft or published status. New records start as drafts and the public API never returns drafts.'],
        ['`revisions`', 'no', 'Keep the last 50 saved versions of each record. Required when any field is `localized`.'],
      ),
      note("These field ids are reserved because the storage layer adds them to every record: `id`, `version`, `created_at`, `updated_at` and `locale`."),
      h2('YAML tips'),
      ul(
        "The compact flow style `- {id: title, label: Title, kind: text}` keeps one field per line; use the expanded block style when a field has many options.",
        "Comments are welcome and are the best place to explain why a field exists.",
        "Commit the files. A content type file is part of your data model.",
      ),
      tip("The complete reference is the header comment of `internal/contenttypes/definition.go` in the repository. Everything on these pages is derived from it."),
    ),

    page(
      'fields',
      'Fields',
      "Every field kind and flag, which flags apply to which kinds, and how each kind appears in the admin and the API.",
      h2('Field keys'),
      table(
        ['Key', 'Applies to', 'Meaning'],
        ['`id`', 'all', 'Identifier, same pattern as the type id. Used as the JSON key.'],
        ['`label`', 'all', 'Label in forms and list columns.'],
        ['`kind`', 'all', 'One of the kinds below.'],
        ['`required`', 'all', 'The record cannot be saved without a value.'],
        ['`unique`', 'all', 'No two records may share the value. Not combinable with `localized`.'],
        ['`options`', '`select`', 'List of allowed values.'],
        ['`to`', '`relation`', 'Id of the target content type.'],
        ['`many`', '`relation`', '`true` lets one record link to several targets.'],
        ['`min`, `max`', '`number`', 'Inclusive numeric bounds.'],
        ['`list`', 'all', 'Show as a column in the admin list. List columns are sortable by clicking the heading.'],
        ['`searchable`', '`text`, `textarea`, `richtext`, `select`', 'Include in the admin list search box and the public API `?search=`.'],
        ['`filterable`', '`text`, `textarea`, `select`, `number`', 'Add a filter to the admin list and allow `?filter[field]=` on the public API.'],
        ['`localized`', '`text`, `textarea`, `richtext`, `media`', 'Keep one value per site locale. See [Localized fields](doc:multi-language/content-locales#localized-fields-in-content-types).'],
      ),
      h2('Field kinds'),
      table(
        ['Kind', 'Stores', 'Admin control', 'API value'],
        ['`text`', 'Short string', 'Single-line input', 'String'],
        ['`textarea`', 'Long plain text', 'Multi-line input', 'String'],
        ['`richtext`', 'Formatted text', 'Rich text editor', 'HTML string'],
        ['`number`', 'Number', 'Number input', 'Number'],
        ['`boolean`', 'True or false', 'Checkbox or switch', 'Boolean'],
        ['`date`', 'Calendar date', 'Date picker', 'Date string'],
        ['`datetime`', 'Date and time', 'Date and time picker', 'Timestamp string'],
        ['`select`', 'One of `options`', 'Dropdown', 'String'],
        ['`media`', 'Reference to a media item', 'Media library picker', 'Media reference'],
        ['`relation`', 'Reference to another record', 'Record picker', 'Record id (array with `many`)'],
      ),
      h2('Examples of each kind'),
      code('yaml', `fields:
  - {id: title,       label: Title,       kind: text,     required: true, unique: true, list: true, searchable: true}
  - {id: summary,     label: Summary,     kind: textarea, searchable: true}
  - {id: body,        label: Body,        kind: richtext}
  - {id: price,       label: Price,       kind: number,   min: 0, max: 10000, list: true, filterable: true}
  - {id: in_stock,    label: In stock,    kind: boolean}
  - {id: released_on, label: Released,    kind: date}
  - {id: starts_at,   label: Starts at,   kind: datetime}
  - {id: status_note, label: Level,       kind: select,   options: [low, medium, high], filterable: true}
  - {id: cover,       label: Cover image, kind: media}
  - {id: related,     label: Related,     kind: relation, to: recipes, many: true}`),
      h2('Validation'),
      ul(
        "`required`, `unique`, `min`, `max` and `options` are enforced by the server on every write, from the admin and the API alike.",
        "Validation failures answer `422` with a `fields` map, for example `{\"title\": \"required\"}`. See [Errors](doc:rest-api/overview#errors).",
        "A field may not be both `localized` and `unique`, `searchable` or `filterable`; the loader rejects such files.",
      ),
      warn("Changing a field's `kind` after records exist is a data migration, not a rename. Add a new field, move the data, then remove the old one."),
    ),

    page(
      'workflow',
      'Relations, drafts and revisions',
      "Link records to each other, add a publishing workflow and keep a history of changes.",
      h2('Relations'),
      p("A `relation` field points at another content type by its id. Use `many: true` for a list of targets."),
      code('yaml', `# content-types/team.yaml  (target)
id: team
label: Team
title_field: name
public: true
fields:
  - {id: name, label: Name, kind: text, required: true, list: true}

# content-types/recipes.yaml  (source)
fields:
  - {id: author, label: Author, kind: relation, to: team}
  - {id: tasters, label: Tasters, kind: relation, to: team, many: true}`),
      p("In the API a relation is the id of the target record; fetch it with `GET /api/v1/content/team/<id>` or list the target type and join on the client."),
      h2('Drafts'),
      p("With `drafts: true` every record has a status field:"),
      table(
        ['Status', 'Visible in the public API', 'How it changes'],
        ['draft', 'No. A draft id answers 404.', 'New records start here. **Move to draft** in the admin returns a record here.'],
        ['published', 'Yes', '**Publish** in the admin.'],
      ),
      p("Without `drafts`, a record is live as soon as it is saved (when the type is `public`)."),
      h2('Revisions'),
      p("With `revisions: true` HydroGo keeps the last 50 saved versions of every record. The edit page lists them; each can be compared with the current record and restored. Restoring saves a new version and keeps the current status, so history is never rewritten."),
      h2('Concurrent editing'),
      p("Records carry a `version` number. An update must send the version it was based on; if someone else saved in between, the server rejects the write with a conflict instead of overwriting their change. The admin shows who else is editing a record."),
      tip("Recommended defaults for editorial content: `drafts: true` and `revisions: true`. For settings-like singletons (site settings, navigation) neither is usually needed."),
    ),

    page(
      'tooling',
      'Tooling and schema',
      "The hydro content-type commands, JSON Schema for editor completion, and the stock types that ship with HydroGo.",
      h2('Create and validate'),
      code('bash', `hydro content-type new recipes          # writes content-types/recipes.yaml
hydro content-type validate             # loads every file and reports errors
hydro content-type new events --dir my-types
hydro content-type validate --dir my-types`),
      table(
        ['Command', 'What it does'],
        ['`hydro content-type new ID [--dir content-types]`', 'Writes a skeleton file for the new type. `ID` follows the id pattern.'],
        ['`hydro content-type validate [--dir content-types]`', 'Loads every `*.yaml` file in the directory and reports the first problem per file: unknown keys, bad kinds, missing relation targets, invalid ids.'],
      ),
      tip("Run `hydro content-type validate` in CI so a broken YAML never reaches a server."),
      h2('Editor completion with JSON Schema'),
      p("A machine-readable schema for content type files is in the repository at `docs/schemas/content-type.schema.json`. Add this line as the first line of a YAML file and editors with the YAML language server (VS Code YAML extension, for instance) give completion, hover docs and inline validation:"),
      code('yaml', "# yaml-language-server: $schema=https://raw.githubusercontent.com/ardaozyildirim/hydrogo/main/docs/schemas/content-type.schema.json"),
      p("Offline, point at a local copy instead: `# yaml-language-server: $schema=../schemas/content-type.schema.json`."),
      h2('Stock content types'),
      p("`hydro new --preset ...` copies stock types into your project. They are examples; edit or delete freely."),
      table(
        ['Type', 'Purpose'],
        ['`site_settings`', 'Brand name, tagline and footer texts (a single record).'],
        ['`site_navigation`', 'Header and footer menu links.'],
        ['`faq`', 'Questions and answers with category.'],
        ['`team`', 'People with localized role and bio.'],
        ['`testimonials`', 'Customer quotes.'],
        ['`comments`', 'Reader comments that wait for moderation (draft to published).'],
        ['`galleries`', 'Photo galleries.'],
        ['`homepage_sections`, `masthead`, `columnists`, `official_announcements`, `ads`, `polls`, `videos`', 'Building blocks of the news portal preset.'],
      ),
      note("Some news-portal stock types ship with Turkish labels because they were made for a Turkish news site. Edit the `label` and `description` values to your language."),
      h2('Admin API for content types'),
      p("The admin lists and manages content types at `/api/v1/admin/content-types`, and records at `/api/v1/admin/content/{type}`. All of them need an authenticated credential; see [Admin API](doc:rest-api/admin-api)."),
    ),
  ],
};
