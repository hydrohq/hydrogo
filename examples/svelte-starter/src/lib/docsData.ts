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
      lead: 'HydroGo is a headless, single-binary Content Management System engineered for speed, minimalism, and developer freedom.',
      sections: [
        {
          title: 'Architecture',
          id: 'architecture',
          text: 'HydroGo cleanly decouples the content storage and API layer from your frontend. The Go core serves high-performance JSON REST APIs. No HTML or CSS markup exists in the server core, eliminating templating overhead and keeping the runtime ultra-fast and secure.',
        },
        {
          title: 'Core Principles',
          id: 'core-principles',
          text: '1. Single-Binary: No node_modules, Python, or Ruby required in production. One compiled Go binary runs everywhere.\n2. Predictable JSON APIs: Every endpoint follows a consistent {data, meta} envelope.\n3. Declarative Schema: Custom content types are modeled as clean YAML files in content-types/ without requiring custom database migrations or boilerplate Go code.\n4. Database Agnostic: Native first-class support for SQLite (with WAL mode), PostgreSQL, and MySQL.',
        },
        {
          title: 'Headless Design',
          id: 'headless-design',
          text: 'Because HydroGo is strictly headless, you build your customer-facing applications in whatever frontend framework you prefer—Svelte, Next.js, Nuxt, Astro, or native mobile apps (iOS/Android)—simply consuming the standardized REST API.',
        },
      ],
    },
  },

  'quick-start': {
    id: 'quick-start',
    title: 'Quick Start',
    category: 'Getting Started',
    badge: 'Popular',
    description: 'Get a HydroGo instance up and running locally in under two minutes.',
    headings: ['Prerequisites', 'Starting Local Dev Server', 'Connecting Your Frontend'],
    content: {
      lead: 'Start building your content-driven application in minutes.',
      sections: [
        {
          title: 'Prerequisites',
          id: 'prerequisites',
          text: 'You only need Go 1.22+ and Git installed on your system. SQLite is embedded by default, requiring no database installation.',
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
            text: 'The devserver boots HydroGo with seeded demo data, applies migrations automatically, and starts listening on http://127.0.0.1:8090.',
          },
        },
        {
          title: 'Connecting Your Frontend',
          id: 'connecting-your-frontend',
          text: 'You can test the public API immediately with cURL or any HTTP client:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'curl http://127.0.0.1:8090/api/v1/pages\ncurl http://127.0.0.1:8090/api/v1/news',
          },
        },
      ],
    },
  },

  configuration: {
    id: 'configuration',
    title: 'Configuration Guide',
    category: 'Getting Started',
    description: 'Configure ports, database drivers, JWT secrets, and CORS origins.',
    headings: ['Configuration File', 'Environment Variables', 'CORS & Security', 'Local and Production Settings'],
    content: {
      lead: 'HydroGo reads configuration from config.yaml with full override support via environment variables.',
      sections: [
        {
          title: 'Configuration File',
          id: 'configuration-file',
          text: 'A clean config.yaml file sets up your server:',
          code: {
            lang: 'yaml',
            filename: 'config.yaml',
            code: `server:
  port: 8090
  bind: "127.0.0.1"
  public_origin: "https://yourdomain.com"

database:
  dialect: sqlite # sqlite | postgres | mysql
  dsn: "hydrogo.db"

security:
  cors_allowed_origins:
    - "https://yourdomain.com"
    - "http://localhost:3000"`,
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
          title: 'CORS & Security',
          id: 'cors-security',
          text: 'Set allowed origins to the exact origins of browser clients, including scheme and port where needed. Keep database credentials and signing keys in protected deployment secrets. Bind the API to a private interface when a reverse proxy is responsible for public TLS termination.',
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
    description: 'Command line utilities for database migrations, user management, and schema generation.',
    headings: ['Core Commands', 'Database Migrations', 'User Management', 'OpenAPI Schema Export'],
    content: {
      lead: 'HydroGo ships with built-in subcommands for operations and automation.',
      sections: [
        {
          title: 'Core Commands',
          id: 'core-commands',
          text: 'List of standard CLI subcommands:',
          table: {
            headers: ['Command', 'Description'],
            rows: [
              ['hydro serve', 'Starts the HTTP API server'],
              ['hydro migrate', 'Runs pending database migrations across all tables'],
              ['hydro user add', 'Creates an administrator or editor user'],
              ['hydro schema openapi', 'Exports the OpenAPI 3.0 specification JSON'],
            ],
          },
        },
        {
          title: 'OpenAPI Schema Export',
          id: 'openapi-schema-export',
          text: 'Export the complete OpenAPI contract to generate client SDKs or TypeScript types:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'hydro schema openapi > openapi.json',
          },
        },
        {
          title: 'Database Migrations',
          id: 'database-migrations',
          text: 'Run pending migrations as part of deployment before starting a new server version. The repository keeps a migration for each supported SQL dialect so the same schema change can be applied on SQLite, PostgreSQL, and MySQL.',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'hydro migrate --config config.yaml',
          },
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
          text: 'Drop a YAML file into content-types/. For example, content-types/faq.yaml:',
          code: {
            lang: 'yaml',
            filename: 'content-types/faq.yaml',
            code: `id: faq
singular: FAQ
plural: FAQs
public: true
drafts: true
fields:
  - id: question
    type: string
    required: true
    searchable: true
  - id: answer
    type: richtext
    required: true
  - id: category
    type: select
    options: ["general", "billing", "technical"]
  - id: sort_order
    type: integer
    default: 0`,
          },
        },
        {
          title: 'Field Options',
          id: 'field-options',
          text: 'Each field has an id and a supported type. Depending on its type, a definition can also include required, default, searchable, localized, options, and validation constraints. Keep IDs stable once content has been entered; they are used in stored records and API responses.',
          code: {
            lang: 'yaml',
            filename: 'Field options',
            code: `fields:
  - id: title
    type: string
    required: true
    searchable: true
  - id: status
    type: select
    options: [draft, published]
    default: draft
  - id: published_at
    type: date
    localized: true`,
          },
        },
        {
          title: 'Admin and Public APIs',
          id: 'admin-public-apis',
          text: 'A definition creates a schema-driven admin collection at /admin/content/{id}. When public is true, published records are also readable from GET /api/v1/content/{id} and GET /api/v1/content/{id}/{record-id}. Drafts and write operations remain behind the admin API and its permissions.',
        },
        {
          title: 'Content Lifecycle',
          id: 'content-lifecycle',
          text: 'Enable drafts when editors need to review work before publication. Schema flags can also enable revisions, list search and filters, and localized fields. Publishing updates the API data immediately; a frontend can fetch the new content without a rebuild.',
        },
        {
          title: 'Automatic API Generation',
          id: 'automatic-api-generation',
          text: 'With public: true, HydroGo immediately serves GET /api/v1/content/faq and GET /api/v1/content/faq/{id} with filtering, sorting, and full-text search.',
        },
      ],
    },
  },

  'field-types': {
    id: 'field-types',
    title: 'Field Types & Validation',
    category: 'Content & Modeling',
    description: 'Comprehensive guide to supported field types and validation rules.',
    headings: ['Supported Field Types', 'Validation and Defaults', 'Localization'],
    content: {
      lead: 'HydroGo supports rich field types tailored for modern content applications.',
      sections: [
        {
          title: 'Supported Field Types',
          id: 'supported-field-types',
          text: 'Standard data types available in collection definitions:',
          table: {
            headers: ['Field Type', 'Storage Type', 'Description'],
            rows: [
              ['string', 'VARCHAR(255)', 'Short single-line text (titles, names, slugs)'],
              ['text', 'TEXT', 'Multi-line plain text (descriptions, notes)'],
              ['richtext', 'JSON / HTML', 'Structured formatting with headings, lists, links, images'],
              ['integer', 'INTEGER', 'Whole numbers (counters, sort orders, ratings)'],
              ['boolean', 'BOOLEAN', 'True/false flags (featured, active, sponsored)'],
              ['select', 'VARCHAR(64)', 'Dropdown selection from a predefined options list'],
              ['media', 'VARCHAR(255)', 'Image or document reference with WebP variants'],
              ['date', 'DATETIME', 'Date or timestamp values'],
            ],
          },
        },
        {
          title: 'Validation and Defaults',
          id: 'validation-and-defaults',
          text: 'Use required to reject missing values and default to provide an initial value when a record is created. Select fields should declare their allowed options. Validation happens when content is saved, so invalid values are reported to the editor instead of silently entering the collection.',
        },
        {
          title: 'Localization',
          id: 'localization',
          text: 'Mark translatable fields as localized when the project has localization enabled. A localized field stores a value for each configured locale; fields that are not localized remain shared across translations. Set the default locale in the i18n configuration.',
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
          text: 'The Pages module manages hierarchical content with slugs, layout templates, and modular block payloads for landing pages and docs.',
        },
        {
          title: 'Choosing a Content Model',
          id: 'choosing-a-content-model',
          text: 'Use pages for slug-addressable site content and structured page blocks; use news for chronological editorial content with publication metadata. Use a YAML content type for repeatable records such as FAQs, locations, or team profiles when they do not need module-specific business logic.',
        },
        {
          title: 'News Module',
          id: 'news-module',
          text: 'The News module handles editorial articles with categories, tags, author metadata, cover media, scheduled publishing dates, and view counts.',
        },
      ],
    },
  },

  'media-library': {
    id: 'media-library',
    title: 'Media Engine & WebP',
    category: 'Content & Modeling',
    description: 'Automatic image processing, WebP variant generation, and BlurHash placeholders.',
    headings: ['Image Processing Pipeline', 'Serving Media'],
    content: {
      lead: 'Zero-configuration media optimization designed for optimal Core Web Vitals.',
      sections: [
        {
          title: 'Image Processing Pipeline',
          id: 'image-processing-pipeline',
          text: 'When images are uploaded, HydroGo automatically creates modern WebP variants (thumbnail, medium, large), calculates focal point coordinates, and produces compact BlurHash placeholder strings for smooth UI loading.',
        },
        {
          title: 'Serving Media',
          id: 'serving-media',
          text: 'Uploaded media is served under /media/{id} with aggressive Cache-Control: public, max-age=31536000, immutable headers for maximum CDN and browser caching efficiency.',
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
  "data": [
    {
      "id": "item_123",
      "title": "Welcome to HydroGo",
      "slug": "welcome"
    }
  ],
  "meta": {
    "total": 1,
    "limit": 20,
    "offset": 0
  }
}`,
          },
        },
        {
          title: 'Pagination Metadata',
          id: 'pagination-metadata',
          text: 'List endpoints accept limit and offset for paging through results. Responses include metadata when the endpoint provides a total or pagination details. Locale-aware endpoints accept locale; use the documented query parameters for each resource rather than assuming every collection supports identical filters.',
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
  "request_id": "req_8f93a1c2"
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
    description: 'Fetch published site pages by slug or list all site hierarchy.',
    headings: ['List Pages', 'Get Page By Slug', 'Locale and Preview'],
    content: {
      lead: 'Endpoints to query static pages and layout structures.',
      sections: [
        {
          title: 'List Pages',
          id: 'list-pages',
          text: 'Fetch all published pages in the current locale:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages"',
          },
        },
        {
          title: 'Get Page By Slug',
          id: 'get-page-by-slug',
          text: 'Retrieve a page by its URL slug:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages/about"',
          },
        },
        {
          title: 'Locale and Preview',
          id: 'locale-and-preview',
          text: 'The list endpoint accepts locale to return content for a language. For a public page lookup by slug, use the documented by-slug query route. Draft preview is a separate protected flow that uses a signed preview token; never expose that token in public links or logs.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages?locale=en"\ncurl "http://127.0.0.1:8090/api/v1/pages/by-slug?locale=en&slug=about"',
          },
        },
      ],
    },
  },

  'news-api': {
    id: 'news-api',
    title: 'News & Articles Endpoints',
    category: 'REST API Reference',
    description: 'Query published articles with filtering, search, and pagination.',
    headings: ['List Articles', 'Article Lookup', 'Filtering and Pagination'],
    content: {
      lead: 'Query published news articles and blog posts.',
      sections: [
        {
          title: 'List Articles',
          id: 'list-articles',
          text: 'Fetch published articles ordered chronologically:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/news?limit=10&offset=0"',
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
          title: 'Filtering and Pagination',
          id: 'filtering-and-pagination',
          text: 'List requests support locale and pagination with limit. Use only filters documented by the API contract for your HydroGo version; endpoints can expose different query parameters.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/news?search=release&category=product&featured=true"',
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
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/content/faq?search=billing&limit=10"',
          },
        },
      ],
    },
  },

  'auth-and-tokens': {
    id: 'auth-and-tokens',
    title: 'Authentication & Tokens',
    category: 'REST API Reference',
    description: 'Authenticate API requests using Bearer tokens, API keys, or session cookies.',
    headings: ['Bearer Token Authentication', 'API Keys'],
    content: {
      lead: 'HydroGo supports three authentication methods: Bearer tokens, API keys, and session cookies.',
      sections: [
        {
          title: 'Bearer Token Authentication',
          id: 'bearer-token-authentication',
          text: 'Pass an API token in the Authorization header for authenticated operations:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl -H "Authorization: Bearer <your-token>" http://127.0.0.1:8090/api/v1/admin/me',
          },
        },
        {
          title: 'API Keys',
          id: 'api-keys',
          text: 'For machine-to-machine or headless frontend integration, generate API keys scoped to read-only or preview access in the configuration.',
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
      lead: 'Health endpoints for Docker, Kubernetes, systemd, or Cloudflare probes.',
      sections: [
        {
          title: 'Readiness Probe',
          id: 'readiness-probe',
          text: 'Returns 200 OK ready when the database connection is healthy and all migrations are verified:',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl -i http://127.0.0.1:8090/readyz',
          },
        },
      ],
    },
  },

  'production-binary': {
    id: 'production-binary',
    title: 'Single-Binary Deploy',
    category: 'Production & Ops',
    description: 'Compile and run a production HydroGo server on Linux, macOS, or Windows.',
    headings: ['Compiling', 'Systemd Service'],
    content: {
      lead: 'Deploy HydroGo as a single self-contained binary without external runtimes or node_modules in production.',
      sections: [
        {
          title: 'Compiling',
          id: 'compiling',
          text: 'Cross-compile a standalone static binary with Go:',
          code: {
            lang: 'bash',
            filename: 'Terminal',
            code: 'GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/hydro ./cmd/hydro',
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
    description: 'Production tuning for SQLite WAL mode, PostgreSQL connection pooling, and MySQL 8+.',
    headings: ['SQLite (Zero-Config)', 'PostgreSQL & MySQL'],
    content: {
      lead: 'Choose the storage engine that fits your project scale.',
      sections: [
        {
          title: 'SQLite (Zero-Config)',
          id: 'sqlite-zero-config',
          text: 'HydroGo configures SQLite with WAL (Write-Ahead Logging), busy timeouts, and concurrent read connections, capable of handling hundreds of requests per second on a simple VPS.',
        },
        {
          title: 'PostgreSQL & MySQL',
          id: 'postgresql-mysql',
          text: 'For multi-instance clustering or high write concurrency, switch database.dialect to postgres or mysql in config.yaml without altering any of your application code.',
        },
      ],
    },
  },
};
