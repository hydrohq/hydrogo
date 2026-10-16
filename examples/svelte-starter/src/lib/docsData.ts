export interface DocTopic {
  id: string;
  title: string;
  category: string;
  badge?: string;
  description: string;
  headings: string[];
  content: {
    lead: string;
    sections: Array<{
      title: string;
      id: string;
      text: string;
      callout?: {
        type: 'note' | 'tip' | 'warning';
        text: string;
      };
      code?: {
        lang: string;
        filename?: string;
        code: string;
      };
      table?: {
        headers: string[];
        rows: string[][];
      };
    }>;
  };
}

export interface DocCategory {
  title: string;
  topics: Array<{
    id: string;
    title: string;
    badge?: string;
  }>;
}

export const docCategories: DocCategory[] = [
  {
    title: 'Getting Started',
    topics: [
      { id: 'overview', title: 'Overview & Philosophy' },
      { id: 'quick-start', title: 'Quick Start', badge: 'Popular' },
      { id: 'configuration', title: 'Configuration Guide' },
      { id: 'cli-commands', title: 'CLI Reference' },
    ],
  },
  {
    title: 'Content & Modeling',
    topics: [
      { id: 'content-types', title: 'YAML Content Types', badge: 'Core' },
      { id: 'field-types', title: 'Field Types & Validation' },
      { id: 'pages-and-news', title: 'Pages & News Modules' },
      { id: 'media-library', title: 'Media Engine & WebP' },
    ],
  },
  {
    title: 'REST API Reference',
    topics: [
      { id: 'api-overview', title: 'API Architecture & Envelope' },
      { id: 'pages-api', title: 'Pages Endpoints' },
      { id: 'news-api', title: 'News & Articles Endpoints' },
      { id: 'custom-content-api', title: 'Custom Content Endpoints' },
      { id: 'auth-and-tokens', title: 'Authentication & Tokens' },
      { id: 'system-probes', title: 'Health Probes (/readyz)' },
    ],
  },
  {
    title: 'Production & Ops',
    topics: [
      { id: 'production-binary', title: 'Single-Binary Deploy' },
      { id: 'reverse-proxy', title: 'Reverse Proxy (Nginx/Caddy)' },
      { id: 'database-setup', title: 'SQLite, Postgres & MySQL' },
      { id: 'site-and-forms', title: 'Site API & Forms' },
    ],
  },
];

export const docTopics: Record<string, DocTopic> = {
  overview: {
    id: 'overview',
    title: 'Overview & Philosophy',
    category: 'Getting Started',
    badge: 'Headless v2.0',
    description: 'Learn the architectural principles behind HydroGo single-binary headless CMS.',
    headings: ['Architecture', 'Core Principles', 'Headless Design'],
    content: {
      lead: 'HydroGo is a single-binary headless Go CMS. It stores and serves content through a JSON API; each project builds its own public frontend against that API.',
      sections: [
        {
          title: 'Architecture',
          id: 'architecture',
          text: 'The server owns content models, publishing workflows, media, members, and the REST API. It does not render a public website. The embedded admin is a Svelte single-page app, while your public site can use any framework or native client.',
        },
        {
          title: 'Core Principles',
          id: 'core-principles',
          text: 'HydroGo keeps content changes live in the database, exposes JSON rather than server-rendered pages, and lets simple collections come from YAML schemas. The core supports SQLite, PostgreSQL, and MySQL. API responses use a shared data/meta success envelope and a request_id-bearing error envelope.',
        },
        {
          title: 'Headless Design',
          id: 'headless-design',
          text: 'The included Svelte starter is an example API consumer, not a required frontend. A frontend can fetch published content directly or through its own backend-for-frontend when credentials or server-side caching are needed.',
        },
      ],
    },
  },

  'quick-start': {
    id: 'quick-start',
    title: 'Quick Start',
    category: 'Getting Started',
    badge: 'Popular',
    description: 'Run the CMS and starter locally, then make your first public API requests.',
    headings: ['Prerequisites', 'Starting Local Dev Server', 'Connecting Your Frontend'],
    content: {
      lead: 'Start building your content-driven application in minutes.',
      sections: [
        {
          title: 'Prerequisites',
          id: 'prerequisites',
          text: 'Use the Go version declared in go.mod (Go 1.27) and Git. The devserver supplies the local development workflow and demo data; production database and media storage still need persistent locations.',
        },
        {
          title: 'Starting Local Dev Server',
          id: 'starting-local-dev-server',
          text: 'Clone the repository and run the development server script:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'git clone https://github.com/ozyildirims/hydrogo.git\ncd hydrogo\ngo run ./tools/devserver',
          },
          callout: {
            type: 'tip',
          text: 'The repository devserver runs the API and Vite with HMR for the admin and starter development workflow, and listens at http://127.0.0.1:8090.',
          },
        },
        {
          title: 'Connecting Your Frontend',
          id: 'connecting-your-frontend',
          text: 'You can test the public API immediately with cURL or any HTTP client:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages?locale=en"\ncurl "http://127.0.0.1:8090/api/v1/news?locale=en"',
          },
        },
      ],
    },
  },

  configuration: {
    id: 'configuration',
    title: 'Configuration Guide',
    category: 'Getting Started',
    description: 'Set the HTTP listener, database, media storage, security, and enabled modules.',
    headings: ['Configuration File', 'Environment Variables', 'Modules and Profiles', 'Security and Origins', 'Local and Production Settings'],
    content: {
      lead: 'HydroGo reads configuration from config.yaml with full override support via environment variables.',
      sections: [
        {
          title: 'Configuration File',
          id: 'configuration-file',
          text: 'Configuration follows the shipped config.example.yaml structure. Keep deployment secrets out of source control:',
          code: {
            lang: 'yaml',
            filename: 'config.yaml',
            code: `profile: compact
modules: [core, auth, members, pages, media, site, news, collections]
http:
  address: "127.0.0.1:8080"
  public_origin: "https://cms.example.com"

database:
  dialect: sqlite # sqlite | postgres | mysql
  dsn: "hydrogo.db"

media:
  dir: "data/media"
security:
  secure_cookies: true
  enable_hsts: true
  headers_preset: strict
  public_api_origins: []`,
          },
        },
        {
          title: 'Environment Variables',
          id: 'environment-variables',
          text: 'Environment variables are useful for deployment secrets and host-specific settings. HydroGo reads selected settings such as the HTTP address, database dialect and DSN, media directory, and content-type directory from HYDRO_ variables. Check the shipped config.example.yaml and internal/config for the exact supported names; not every YAML key has a matching environment override.',
          code: {
            lang: 'bash',
            filename: 'Environment',
            code: 'HYDRO_HTTP_ADDRESS=127.0.0.1:8090\nHYDRO_DATABASE_DIALECT=postgres\nHYDRO_DATABASE_DSN="postgres://user:password@localhost:5432/hydro?sslmode=disable"',
          },
        },
        {
          title: 'Modules and Profiles',
          id: 'modules-and-profiles',
          text: 'The modules list controls which capabilities are active. Keep it intentional: inactive modules should not add routes, migrations, workers, assets, or admin navigation. Profiles select a capacity preset; consult the config reference and validate the configuration before deployment.',
        },
        {
          title: 'Security and Origins',
          id: 'security-and-origins',
          text: 'For HTTPS deployments, enable secure cookies and HSTS. public_api_origins is an optional explicit allowlist for credential-free published API reads; wildcard origins are not supported. Browser writes should use same-site sessions or a server-side BFF. Do not put admin credentials or private API keys in a browser bundle.',
        },
        {
          title: 'Local and Production Settings',
          id: 'local-production-settings',
          text: 'For local development, the starter devserver uses loopback and SQLite. In production, provide a persistent media directory and database DSN, configure a stable public origin, and run the binary behind HTTPS. Use config.example.yaml as the canonical list of configuration sections.',
        },
      ],
    },
  },

  'cli-commands': {
    id: 'cli-commands',
    title: 'CLI Reference',
    category: 'Getting Started',
    description: 'Commands for running, validating, migrating, backing up, and maintaining HydroGo.',
    headings: ['Run and Validate', 'Database and Schema', 'Admin and SDK Commands', 'Backup and Restore'],
    content: {
      lead: 'HydroGo ships with built-in subcommands for operations and automation.',
      sections: [
        {
          title: 'Run and Validate',
          id: 'run-and-validate',
          text: 'Common commands for local development and startup checks:',
          table: {
            headers: ['Command', 'Description'],
            rows: [
              ['hydro serve', 'Starts the configured HTTP API and embedded admin'],
              ['hydro run / hydro dev', 'Runs the development workflow with dependency options'],
              ['hydro validate', 'Validates configuration and runtime prerequisites'],
              ['hydro doctor', 'Inspects operational health; optional --fix is explicit'],
              ['hydro healthcheck', 'Runs the process health check'],
            ],
          },
        },
        {
          title: 'Database and Schema',
          id: 'database-and-schema',
          text: 'Migrations apply pending database changes. Schema diff compares release contracts; it is not an OpenAPI export command.',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'hydro migrate --config config.yaml\nhydro schema diff --baseline RELEASE.json --current RELEASE.json --public-key KEY',
          },
        },
        {
          title: 'Admin and SDK Commands',
          id: 'admin-and-sdk-commands',
          text: 'Create the first administrator from a password supplied on stdin. Generate SDKs from the API contract with the dedicated generator:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'hydro create-admin --email admin@example.com --password-stdin\nhydro sdk generate --lang typescript --out ./generated',
          },
        },
        {
          title: 'Backup and Restore',
          id: 'backup-and-restore',
          text: 'The CLI includes database backup, pruning, verification, and restore workflows. Read each command help and use a staging destination before recovery; restore commands require explicit safety flags.',
          code: { lang: 'bash', filename: 'Terminal', code: 'hydro backup sqlite --destination ./backup\nhydro restore verify --snapshot ./backup' },
        },
      ],
    },
  },

  'content-types': {
    id: 'content-types',
    title: 'YAML Content Types',
    category: 'Content & Modeling',
    badge: 'Core',
    description: 'Define custom content collections using declarative YAML files.',
    headings: ['Defining Collections', 'Field Options', 'Admin and Public APIs', 'Content Lifecycle'],
    content: {
      lead: 'Define structured content types in seconds without writing SQL migrations or backend Go structs.',
      sections: [
        {
          title: 'Defining Collections',
          id: 'defining-collections',
          text: 'Add a YAML definition under content-types/. This valid example creates a public FAQ collection with drafts, revisions, searchable text, and list filters:',
          code: {
            lang: 'yaml',
            filename: 'content-types/faq.yaml',
            code: `id: faq
label: FAQ
description: Questions and answers shown on the site.
title_field: question
public: true
drafts: true
revisions: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
  - {id: answer, label: Answer, kind: richtext, required: true, searchable: true}
  - {id: category, label: Category, kind: select, options: [general, account, billing], list: true, filterable: true}
  - {id: sort_order, label: Order, kind: number, list: true}`,
          },
        },
        {
          title: 'Field Options',
          id: 'field-options',
          text: 'Fields use kind (not type). Common options are required, unique, options for select values, to/many for relations, min/max for numbers, and list/searchable/filterable for admin and public discovery. Keep IDs stable after records exist because they are part of stored data and API responses.',
          code: {
            lang: 'yaml',
            filename: 'Field options',
            code: `fields:
  - id: title
    label: Title
    kind: text
    required: true
    searchable: true
    list: true
  - id: status
    label: Status
    kind: select
    options: [active, archived]
    filterable: true
  - id: related_page
    label: Related page
    kind: relation
    to: pages
    many: false`,
          },
        },
        {
          title: 'Admin and Public APIs',
          id: 'admin-public-apis',
          text: 'A definition creates an admin collection at /admin/content/{id}. With public: true, published records are readable at GET /api/v1/content/{id} and GET /api/v1/content/{id}/{record-id}. Drafts and write operations use the protected admin API.',
        },
        {
          title: 'Content Lifecycle',
          id: 'content-lifecycle',
          text: 'With drafts enabled, new records start as drafts and the public API excludes unpublished records. Revisions retain up to 50 saved versions; restoring one creates a new version. Localized fields require revisions and configured locales, and cannot be unique, searchable, or filterable. Publishing changes database/API output immediately without rebuilding a frontend.',
        },
        {
          title: 'Querying Public Content',
          id: 'querying-public-content',
          text: 'Lists accept limit (default 25, maximum 100), offset, sort (prefix a field with - for descending), search across searchable fields, and exact filter[field] matches for filterable fields. Responses include has_more rather than a universal total count.',
          code: { lang: 'bash', filename: 'cURL', code: 'curl "http://127.0.0.1:8090/api/v1/content/faq?limit=10&search=account&filter[category]=account"' },
        },
      ],
    },
  },

  'field-types': {
    id: 'field-types',
    title: 'Field Types & Validation',
    category: 'Content & Modeling',
    description: 'Available schema field kinds, constraints, relations, and localization rules.',
    headings: ['Supported Field Kinds', 'Validation and Relations', 'Localization'],
    content: {
      lead: 'HydroGo supports rich field types tailored for modern content applications.',
      sections: [
        {
          title: 'Supported Field Kinds',
          id: 'supported-field-types',
          text: 'Standard data types available in collection definitions:',
          table: {
            headers: ['Field Type', 'Storage Type', 'Description'],
            rows: [
              ['text', 'Text', 'Single-line text values'],
              ['textarea', 'Text', 'Multi-line plain text'],
              ['richtext', 'Rich text', 'Formatted editorial content'],
              ['number', 'Number', 'Numeric values; supports min/max'],
              ['boolean', 'Boolean', 'True/false values'],
              ['date / datetime', 'Date/time', 'Calendar date or date-time value'],
              ['select', 'Option', 'One value from declared options'],
              ['media', 'Media reference', 'Reference to a media asset'],
              ['relation', 'Relation', 'Reference to another collection; to/many configure target and cardinality'],
            ],
          },
        },
        {
          title: 'Validation and Relations',
          id: 'validation-and-defaults',
          text: 'Use required to reject missing values, unique to prevent duplicate values, min/max on numbers, and options on select fields. Relations use to for their target and many to allow multiple related records. Validation runs when records are written.',
        },
        {
          title: 'Localization',
          id: 'localization',
          text: 'Only text, textarea, richtext, and media fields can be localized. Localized fields need revisions enabled on the content type and cannot also be unique, searchable, or filterable. Configure available and default locales in i18n; public reads can request a locale, with localized values falling back per field to the default locale.',
        },
      ],
    },
  },

  'pages-and-news': {
    id: 'pages-and-news',
    title: 'Pages & News Modules',
    category: 'Content & Modeling',
    description: 'Built-in core modules for static pages and timestamped editorial news.',
    headings: ['Pages Module', 'News Module', 'Choosing a Content Model'],
    content: {
      lead: 'Built-in publishing systems for marketing pages and chronological blog posts.',
      sections: [
        {
          title: 'Pages Module',
          id: 'pages-module',
          text: 'Pages store locale-aware, slug-addressable published content and structured blocks. The public list returns summaries; fetch a page by locale and slug to get the page payload. Preview links are private, short-lived, and tied to a draft revision.',
        },
        {
          title: 'Choosing a Content Model',
          id: 'choosing-a-content-model',
          text: 'Use pages for slug-addressable site content and structured page blocks; use news for chronological editorial content with publication metadata. Use a YAML content type for repeatable records such as FAQs, locations, or team profiles when they do not need module-specific business logic.',
        },
        {
          title: 'News Module',
          id: 'news-module',
          text: 'News records include title, body, slug, locale, cover media, excerpt, key takeaways, category, author, featured flag, tags, SEO fields, publication status, and optional schedule/expiry timestamps. Public endpoints expose published articles only. The list endpoint supports locale and limit; it does not provide category/search/offset query filters.',
        },
      ],
    },
  },

  'media-library': {
    id: 'media-library',
    title: 'Media Library',
    category: 'Content & Modeling',
    description: 'Upload rules, stored asset metadata, derivative processing, and delivery.',
    headings: ['Uploads and Asset Metadata', 'Derivatives and Storage', 'Serving Media'],
    content: {
      lead: 'Zero-configuration media optimization designed for optimal Core Web Vitals.',
      sections: [
        {
          title: 'Image Processing Pipeline',
          id: 'image-processing-pipeline',
          text: 'The media library accepts JPEG, PNG, WebP, and SVG images, plus MP4, WebM, and QuickTime video. Asset metadata can include dimensions, focal point, dominant colors, blur hash, tags, folder, status, and visibility. Validate size and allowed media at upload boundaries in your client workflow.',
        },
        {
          title: 'Derivatives and Storage',
          id: 'serving-media',
          text: 'Media is stored under the configured media directory or an optional S3-compatible backend. Image/PDF/video derivative jobs use a bounded background queue; derivative availability is asynchronous. Keep the data directory persistent and back it up with the database.',
        },
        {
          title: 'Serving Media',
          id: 'serving-media',
          text: 'Use the asset URL and metadata returned by the media API. Do not assume every source file has a particular WebP size variant; clients should use the original or a derivative that is actually present.',
        },
      ],
    },
  },

  'api-overview': {
    id: 'api-overview',
    title: 'API Architecture & Envelope',
    category: 'REST API Reference',
    description: 'Consistent JSON responses, standard envelopes, and predictable error formats.',
    headings: ['Success Envelope', 'Pagination Metadata', 'Error Envelope'],
    content: {
      lead: 'All HydroGo API responses adhere to a unified, standardized envelope.',
      sections: [
        {
          title: 'Success Envelope',
          id: 'success-envelope',
          text: 'Standard response for single objects or paginated collections:',
          code: {
            lang: 'json',
            filename: 'Response 200 OK',
            code: `{
  "data": [{ "id": "item_123", "title": "Welcome" }],
  "meta": { "limit": 20, "has_more": false }
}`,
          },
        },
        {
          title: 'Pagination Metadata',
          id: 'pagination-metadata',
          text: 'Pagination metadata varies by endpoint. Page and news lists expose limit and has_more; public content types additionally accept offset. Do not rely on a total count unless a specific endpoint documents one. Locale and filtering support also vary by resource.',
        },
        {
          title: 'Error Envelope',
          id: 'error-envelope',
          text: 'Errors consistently include a request_id for end-to-end tracing:',
          code: {
            lang: 'json',
            filename: 'Response 400 Bad Request',
            code: `{
  "code": "VALIDATION_FAILED",
  "message": "The title field is required",
  "request_id": "req_8f93a1c2",
  "fields": { "title": "required" }
}`,
          },
        },
      ],
    },
  },

  'pages-api': {
    id: 'pages-api',
    title: 'Pages Endpoints',
    category: 'REST API Reference',
    description: 'List page summaries, fetch a published page by locale and slug, or preview a draft.',
    headings: ['List Pages', 'Get Page By Slug', 'Locale and Preview'],
    content: {
      lead: 'Endpoints to query static pages and layout structures.',
      sections: [
        {
          title: 'List Pages',
          id: 'list-pages',
          text: 'List published page summaries. The locale is optional and limit defaults to 20 (maximum 100). The response includes pagination metadata with has_more; it is not the full page body.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages?locale=en&limit=20"',
          },
        },
        {
          title: 'Get Page By Slug',
          id: 'get-page-by-slug',
          text: 'Fetch the published page payload using the locale and slug query parameters (or the locale/slug path form):',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages/by-slug?locale=en&slug=about"',
          },
        },
        {
          title: 'Locale and Preview',
          id: 'locale-and-preview',
          text: 'A protected admin request creates a preview link. Its token is bound to the current draft revision and expires when that revision changes. Preview responses are private and must not be cached or exposed as public content.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'GET /api/v1/pages/{id}/preview?token=<preview-token>\nPOST /api/v1/admin/pages/{id}/preview-link',
          },
        },
      ],
    },
  },

  'news-api': {
    id: 'news-api',
    title: 'News & Articles Endpoints',
    category: 'REST API Reference',
    description: 'List published articles and look them up by locale and slug.',
    headings: ['List Articles', 'Article Lookup', 'Draft Preview and Query Limits'],
    content: {
      lead: 'Query published news articles and blog posts.',
      sections: [
        {
          title: 'List Articles',
          id: 'list-articles',
          text: 'Locale is required. The list returns published articles; limit defaults to 20 and is capped at 100:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/news?locale=en&limit=10"',
          },
        },
        {
          title: 'Article Lookup',
          id: 'article-lookup',
          text: 'Resolve a published article by slug with the by-slug endpoint. This query form is useful when the slug comes from a route parameter:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/news/by-slug?locale=en&slug=release-notes"',
          },
        },
        {
          title: 'Draft Preview and Query Limits',
          id: 'draft-preview-and-query-limits',
          text: 'The public list does not accept offset, search, category, or featured filters. Draft previews use a revision-bound token created through the protected admin endpoint. Keep preview links private.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'POST /api/v1/admin/news/{id}/preview-link\nGET /api/v1/news/{id}/preview?token=<preview-token>',
          },
        },
      ],
    },
  },

  'custom-content-api': {
    id: 'custom-content-api',
    title: 'Custom Content Endpoints',
    category: 'REST API Reference',
    description: 'Query any content type defined in content-types/ via public read endpoints.',
    headings: ['Querying Collections'],
    content: {
      lead: 'All YAML content collections with public: true are instantly exposed as REST resources.',
      sections: [
        {
          title: 'Querying Collections',
          id: 'querying-collections',
          text: 'Filter, search, and paginate through custom content items effortlessly:',
          code: { lang: 'bash', filename: 'cURL', code: 'curl "http://127.0.0.1:8090/api/v1/content/faq?search=billing&limit=10&filter[category]=billing"\ncurl "http://127.0.0.1:8090/api/v1/content/faq/RECORD_ID"' },
        },
      ],
    },
  },

  'auth-and-tokens': {
    id: 'auth-and-tokens',
    title: 'Authentication & Tokens',
    category: 'REST API Reference',
    description: 'Authenticate API requests using Bearer tokens, API keys, or session cookies.',
    headings: ['Session Authentication', 'Personal Bearer Tokens', 'API Keys and Scope'],
    content: {
      lead: 'HydroGo supports three authentication methods: Bearer tokens, API keys, and session cookies.',
      sections: [
        {
          title: 'Session Authentication',
          id: 'session-authentication',
          text: 'Admin browser sessions use the HTTP-only hydro_session cookie. Obtain a CSRF token from /csrf-token and send it in X-CSRF-Token for state-changing browser requests. The shared auth middleware also accepts bearer tokens and API keys.',
          code: { lang: 'bash', filename: 'cURL', code: 'curl -c cookies.txt http://127.0.0.1:8090/csrf-token' },
        },
        {
          title: 'Personal Bearer Tokens',
          id: 'personal-bearer-tokens',
          text: 'Personal tokens are created and revoked from the admin token settings/API. They are shown once and stored hashed. Send the token in the Authorization header; these tokens carry admin scope, so never ship them to an untrusted browser.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl -H "Authorization: Bearer hyd_<token>" http://127.0.0.1:8090/api/v1/admin/me',
          },
        },
        {
          title: 'API Keys and Scope',
          id: 'api-keys-and-scope',
          text: 'API keys are sent using X-API-Key and can be limited to read or preview scopes. Use them from trusted server-side code and rotate/revoke them through the admin controls. Keep secrets in deployment configuration, not frontend source.',
          code: { lang: 'bash', filename: 'cURL', code: 'curl -H "X-API-Key: <read-or-preview-key>" http://127.0.0.1:8090/api/v1/pages?locale=en' },
        },
      ],
    },
  },

  'system-probes': {
    id: 'system-probes',
    title: 'Health Probes (/readyz)',
    category: 'REST API Reference',
    description: 'Readiness and liveness checks for load balancers and container orchestrators.',
    headings: ['Readiness Probe'],
    content: {
      lead: 'Health and readiness endpoints for process and dependency checks.',
      sections: [
        {
          title: 'Readiness Probe',
          id: 'readiness-probe',
          text: 'GET /healthz is the liveness check. GET /readyz reports readiness for routing traffic; use the endpoint matching your orchestrator probe. Both are unauthenticated operational endpoints:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl -i http://127.0.0.1:8090/healthz\ncurl -i http://127.0.0.1:8090/readyz',
          },
        },
      ],
    },
  },

  'production-binary': {
    id: 'production-binary',
    title: 'Single-Binary Deploy',
    category: 'Production & Ops',
    description: 'Build and run the API and embedded admin as a Go binary.',
    headings: ['Compiling', 'Systemd Service'],
    content: {
      lead: 'Deploy HydroGo as a single self-contained binary without external runtimes or node_modules in production.',
      sections: [
        {
          title: 'Compiling',
          id: 'compiling',
          text: 'Build the CLI from the repository. Use the supported Go version and platform-specific build configuration; do not assume CGO can be disabled for every database driver:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'go build -trimpath -o bin/hydro ./cmd/hydro',
          },
        },
        {
          title: 'Systemd Service',
          id: 'systemd-service',
          text: 'Run HydroGo under systemd with automatic restart on failure and sandboxed file permissions:',
          code: {
            lang: 'ini',
            filename: '/etc/systemd/system/hydrogo.service',
            code: `[Unit]
Description=HydroGo Headless CMS
After=network.target

[Service]
Type=simple
User=hydrogo
WorkingDirectory=/var/www/hydrogo
ExecStart=/var/www/hydrogo/bin/hydro serve --config /var/www/hydrogo/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target`,
          },
        },
      ],
    },
  },

  'reverse-proxy': {
    id: 'reverse-proxy',
    title: 'Reverse Proxy (Nginx/Caddy)',
    category: 'Production & Ops',
    description: 'Configure Caddy or Nginx with automatic HTTPS and gzip/brotli compression.',
    headings: ['Caddyfile Example', 'Nginx Example'],
    content: {
      lead: 'Expose HydroGo behind modern reverse proxies for automated TLS certificates.',
      sections: [
        {
          title: 'Caddyfile Example',
          id: 'caddyfile-example',
          text: "Caddy provides automated Let's Encrypt certificates in two lines of configuration:",
          code: {
            lang: 'nginx',
            filename: 'Caddyfile',
            code: `cms.yourdomain.com {
  reverse_proxy 127.0.0.1:8090
}`,
          },
        },
        {
          title: 'Nginx Example',
          id: 'nginx-example',
          text: 'Standard Nginx upstream block forwarding headers and requests:',
          code: {
            lang: 'nginx',
            filename: 'nginx.conf',
            code: `server {
  server_name cms.yourdomain.com;

  location / {
    proxy_pass http://127.0.0.1:8090;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}`,
          },
        },
      ],
    },
  },

  'database-setup': {
    id: 'database-setup',
    title: 'SQLite, Postgres & MySQL',
    category: 'Production & Ops',
    description: 'Choose one of the supported SQL dialects and configure persistent storage.',
    headings: ['Supported Dialects', 'Persistent Data and Migrations'],
    content: {
      lead: 'Choose the storage engine that fits your project scale.',
      sections: [
        {
          title: 'Supported Dialects',
          id: 'supported-dialects',
          text: 'HydroGo supports SQLite, PostgreSQL, and MySQL. Select the dialect and DSN in configuration. Choose based on your operational needs and validate the exact driver/runtime setup before rollout.',
        },
        {
          title: 'Persistent Data and Migrations',
          id: 'persistent-data-and-migrations',
          text: 'Keep the database and media directory on persistent storage. Apply migrations as part of a controlled deployment, back up the database and media together, and test restore procedures. Each new migration must include SQLite, PostgreSQL, and MySQL dialect files.',
        },
      ],
    },
  },

  'site-and-forms': {
    id: 'site-and-forms',
    title: 'Site API & Forms',
    category: 'Production & Ops',
    description: 'Read public site settings, navigation, redirects, and submit configured forms.',
    headings: ['Site Settings and Navigation', 'Redirects and Sitemaps', 'Forms'],
    content: {
      lead: 'The site module exposes project settings and navigation data for a frontend to render. Forms are available when their module and form definitions are configured.',
      sections: [
        {
          title: 'Site Settings and Navigation',
          id: 'site-settings-and-navigation',
          text: 'Fetch public settings and locale-specific navigation from the site API. The site settings payload is the configured public projection, not the full private admin configuration.',
          code: { lang: 'bash', filename: 'cURL', code: 'curl http://127.0.0.1:8090/api/v1/site/settings\ncurl http://127.0.0.1:8090/api/v1/site/navigation/en' },
        },
        {
          title: 'Redirects and Sitemaps',
          id: 'redirects-and-sitemaps',
          text: 'Published redirects are available from the public site API. Sitemap endpoints include an index and separate page/article maps when the corresponding content is enabled.',
          code: { lang: 'text', filename: 'Routes', code: 'GET /api/v1/site/redirects\nGET /sitemap.xml\nGET /sitemap-index.xml\nGET /sitemap-pages.xml\nGET /sitemap-articles.xml' },
        },
        {
          title: 'Forms',
          id: 'forms',
          text: 'Configured forms can be fetched and submitted through the public forms API. Submission payloads are validated against the form definition; the inbox and moderation workflows are managed through protected admin routes. Enable and configure the forms module before relying on these routes.',
          code: { lang: 'text', filename: 'Routes', code: 'GET /api/v1/forms/{form}\nPOST /api/v1/forms/{form}/submit\nGET /api/v1/forms/{form}/view' },
        },
      ],
    },
  },
};
