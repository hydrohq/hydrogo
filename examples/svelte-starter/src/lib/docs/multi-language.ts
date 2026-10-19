import { type DocSection, page, p, h2, h3, code, tabs, note, tip, warn, table, ul, ol } from './types';

export const multiLanguage: DocSection = {
  id: 'multi-language',
  title: 'Multi-language',
  pages: [
    page(
      'content-locales',
      'Content locales',
      "Publish your site in several languages: the i18n configuration, per-locale pages and news, localized content type fields and the ?locale= parameter.",
      p("HydroGo has two independent language concepts. **Content locales** (this page) are the languages of your site's content. The **admin interface language** is what editors see in the panel; see [Admin interface language](doc:multi-language/admin-language)."),
      h2('Enable locales'),
      code('yaml', `i18n:
  enabled: true
  default_locale: en
  locales: [en, tr, de]`, 'config.yaml'),
      code('bash', "HYDRO_I18N_ENABLED=true HYDRO_DEFAULT_LOCALE=en HYDRO_LOCALES=en,tr,de hydro serve --config config.yaml"),
      ul(
        "`default_locale` must not be empty and must be one of `locales`.",
        "`locales` holds 1 to 64 codes such as `en`, `tr`, `pt-BR` or `zh-Hans`.",
        "Restart the server after changing locales.",
      ),
      h2('Pages and news'),
      p("Pages and news articles are **per-locale documents**: each has a `locale`, and the slug is unique within that locale. Create the English `about` page and a Turkish `hakkimizda` page as two documents (the page editor also offers a **Translate** action to start from an existing page)."),
      code('bash', `curl 'https://cms.example.com/api/v1/pages/by-slug?locale=en&slug=about'
curl 'https://cms.example.com/api/v1/pages/by-slug?locale=tr&slug=hakkimizda'
curl 'https://cms.example.com/api/v1/news?locale=tr&limit=10'`),
      p("Always pass `locale` to the pages and news endpoints. They do not fall back to another locale's document."),
      h2('Localized fields in content types', 'localized-fields-in-content-types'),
      p("In a content type, mark the fields whose text differs per language with `localized: true`. Storage keeps one value per locale."),
      code('yaml', `id: team
label: Team
revisions: true          # required when any field is localized
public: true
fields:
  - {id: name, label: Name, kind: text, required: true, list: true}
  - {id: role, label: Role, kind: text, list: true, localized: true}
  - {id: bio,  label: Bio,  kind: textarea, localized: true}
  - {id: photo, label: Photo, kind: media, localized: true}`, 'content-types/team.yaml'),
      table(
        ['Rule', 'Detail'],
        ['Allowed kinds', '`text`, `textarea`, `richtext`, `media`.'],
        ['Not combinable with', '`unique`, `searchable`, `filterable`.'],
        ['Requires', '`revisions: true` on the type.'],
        ['Default locale', 'The record itself holds the default locale\'s value; other locales are stored beside it.'],
        ['Activation', 'Only when `i18n.enabled` is true **and** more than one locale is configured. Otherwise the flag changes nothing.'],
      ),
      h3('In the admin'),
      p("Records of types with localized fields show **locale tabs**. Edit the default locale first, then switch tabs to fill the others. A **copy from** action seeds a locale from another one."),
      h3('In the API'),
      code('bash', `# default locale
curl 'https://cms.example.com/api/v1/content/team'
# Turkish values; any missing field falls back to the default locale per field
curl 'https://cms.example.com/api/v1/content/team?locale=tr'`),
      p("With `?locale=` the response is flagged with `metadata.locale` and each record gets a `locale` field. **Fallback happens field by field**: if only `role` is translated, `bio` still comes back in the default language. An unknown locale answers `400`."),
      note("Search, filters and sorting always use the default locale's values, even when `?locale=` asks for another."),
      h2('From your frontend'),
      code('ts', `const locale = params.lang ?? 'en';

const page = await hydro.getPage(locale, params.slug);
const news = await hydro.listNews(locale, 10);
const team = await hydro.listContent('team', { locale });`),
      tip("Link translations to each other in your frontend by convention (for example the same page id in your own mapping, or a shared tag). HydroGo does not enforce that a translation exists for every locale."),
      h2('Site-level text'),
      p("Navigation menus and translation strings are stored per locale too: **Site > Navigation** and **Site > Translations** in the admin, and `GET /api/v1/site/navigation/{locale}` in the API."),
    ),

    page(
      'admin-language',
      'Admin interface language',
      "Each person can pick the language of the admin panel; an administrator sets the project default. English and Turkish are included.",
      p("The admin panel is localized. This affects only the **interface** (menus, buttons, messages), never the content you edit or what the public API returns. Supported locales today: `en` (English) and `tr` (Türkçe)."),
      h2('Pick your own language'),
      ol(
        "Open **My Account** from the user menu.",
        "Find the **Language** section.",
        "Choose **English**, **Türkçe**, or **Use the project default**. The change applies immediately. It is saved to your account and remembered on this browser.",
      ),
      note("The account setting is saved with `PUT /api/v1/admin/account/locale`."),
      h2('Set the project default'),
      ol(
        "Sign in as an administrator and open **Site settings**.",
        "On the **General** tab, find **Admin language (project default)**.",
        "Choose **English**, **Türkçe** or **Automatic (browser language)**, then save.",
      ),
      p("The default is used for everyone who has not chosen their own language in My Account. The value is stored in the site setting `admin.default_locale`."),
      table(
        ['Where', 'Label', 'Who', 'Effect'],
        ['My Account, Language', 'My language', 'Each user', 'Overrides the project default for this account.'],
        ['Site settings, General', 'Admin language (project default)', 'Administrators', 'Applies to every user without a personal choice.'],
      ),
      h2('How the language is chosen'),
      ol(
        "The signed-in user's own choice in My Account.",
        "Otherwise the project default from Site settings.",
        "If that is **Automatic**, the language of the browser (`en` or `tr`), and English when the browser language is not supported.",
      ),
      h2('For integrators'),
      table(
        ['Endpoint', 'Purpose'],
        ['`GET /api/v1/admin/i18n/{locale}`', 'Returns the translation catalogue of the admin for `en` or `tr`.'],
        ['`PUT /api/v1/admin/account/locale`', 'Saves the signed-in user\'s language. An empty value means "use the project default".'],
        ['`GET /api/v1/admin/schema`', 'Includes `locales.admin_default`, the project default chosen in Site settings (empty until set).'],
      ),
      tip("Admin language and content locales are separate on purpose: an editor can work in a Turkish interface while writing the English version of a page."),
      warn("Adding a content locale (for example `de`) does not add a German admin interface. The panel's languages ship with HydroGo."),
    ),
  ],
};
