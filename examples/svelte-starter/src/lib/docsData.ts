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
      { id: 'system-probes', title: 'Health & Probes (/readyz)' },
    ],
  },
  {
    title: 'Admin Studio',
    topics: [
      { id: 'admin-overview', title: 'Svelte 5 Architecture' },
      { id: 'admin-rbac', title: 'Role-Based Access Control' },
      { id: 'admin-revisions', title: 'Drafts & Revisions' },
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
  'overview': {
    id: 'overview',
    title: 'Overview & Philosophy',
    category: 'Getting Started',
    badge: 'v2.0 Headless',
    description: 'Learn the architectural principles behind HydroGo single-binary headless CMS.',
    headings: ['Architecture', 'Core Principles', 'Why Go & Svelte 5?'],
    content: {
      lead: 'HydroGo is a headless, single-binary Content Management System engineered for speed, minimalism, and developer freedom.',
      sections: [
        {
          title: 'Architecture',
          id: 'architecture',
          text: 'HydroGo cleanly decouples the content management layer from frontend rendering. The Go server serves strictly JSON APIs and embeds the modern Svelte 5 Admin Studio SPA. No HTML or CSS strings exist in the Go core.',
          callout: {
            type: 'note',
            text: 'HydroGo does not render a public website on the server. Your web apps, mobile apps, or static frontends query the REST API independently.',
          },
        },
        {
          title: 'Core Principles',
          id: 'core-principles',
          text: 'Every feature in HydroGo adheres to four strict standards: Single-binary deployment, Tri-dialect database compatibility (SQLite, PostgreSQL, MySQL), zero-code YAML content extensibility, and standard JSON envelope responses.',
          code: {
            lang: 'bash',
            filename: 'terminal',
            code: '# Start HydroGo devserver with rebuild-on-save and embedded SPA\ngo run ./tools/devserver',
          },
        },
        {
          title: 'Why Go & Svelte 5?',
          id: 'why-go-and-svelte-5',
          text: 'Go provides ultra-fast startup (<20ms), tiny memory footprints (<30MB), and reliable concurrency. Svelte 5 runes deliver zero-runtime reactivity and sub-millisecond admin UI rendering.',
        },
      ],
    },
  },

  'quick-start': {
    id: 'quick-start',
    title: 'Quick Start',
    category: 'Getting Started',
    badge: '5 min',
    description: 'Get your local development environment running in under 5 minutes.',
    headings: ['Prerequisites', 'Starting Local Dev Server', 'Accessing Admin Studio', 'Connecting Your Frontend'],
    content: {
      lead: 'Start building immediately with the unified development server, pre-seeded with disposable demo content.',
      sections: [
        {
          title: 'Prerequisites',
          id: 'prerequisites',
          text: 'You need Go 1.27+ and Node.js 18+ installed on your development machine.',
          table: {
            headers: ['Tool', 'Version', 'Purpose'],
            rows: [
              ['Go', '1.27.2+', 'Backend runtime & binary compiler'],
              ['Node.js', '18.x or 20.x+', 'Admin panel and starter frontend toolchain'],
              ['SQLite / Postgres / MySQL', 'Any modern version', 'Database store (SQLite is zero-config)'],
            ],
          },
        },
        {
          title: 'Starting Local Dev Server',
          id: 'starting-local-dev-server',
          text: 'The unified devserver compiles Go on save, proxies Vite HMR for Admin and Frontend, and exposes everything through a single proxy port (8090).',
          code: {
            lang: 'bash',
            filename: 'terminal',
            code: 'go run ./tools/devserver',
          },
          callout: {
            type: 'tip',
            text: 'Pass -address 0.0.0.0:8090 to test the admin studio or starter site from your mobile phone on the same Wi-Fi network.',
          },
        },
        {
          title: 'Accessing Admin Studio',
          id: 'accessing-admin-studio',
          text: 'Open http://127.0.0.1:8090/admin in your browser. Use your local administrator credentials to sign in and explore the pre-seeded content.',
        },
        {
          title: 'Connecting Your Frontend',
          id: 'connecting-your-frontend',
          text: 'Your frontend interacts with the REST API at /api/v1/*. See examples/svelte-starter for an idiomatic Svelte 5 implementation.',
        },
      ],
    },
  },

  'configuration': {
    id: 'configuration',
    title: 'Configuration Guide',
    category: 'Getting Started',
    description: 'Configure database connections, security headers, CORS, media storage, and capacity profiles.',
    headings: ['Configuration Structure', 'Database Dialects', 'Security & Sessions'],
    content: {
      lead: 'HydroGo reads configuration from a single declarative YAML file or environment variable overrides.',
      sections: [
        {
          title: 'Configuration Structure',
          id: 'configuration-structure',
          text: 'A standard config.yaml controls server ports, storage engines, session lifetimes, and CORS rules.',
          code: {
            lang: 'yaml',
            filename: 'config.yaml',
            code: `http:
  listen: "127.0.0.1:8090"
  public_origin: "https://yourdomain.com"
database:
  dialect: "sqlite" # sqlite, postgres, or mysql
  dsn: "data/hydrogo.db"
security:
  secure_cookies: true
  session_ttl: "168h" # 7 days
media:
  dir: "data/media"
  max_upload_bytes: 10485760 # 10MB`,
          },
        },
        {
          title: 'Database Dialects',
          id: 'database-dialects',
          text: 'HydroGo supports SQLite, PostgreSQL, and MySQL out of the box with zero code changes. Every migration is tested across all three dialects.',
        },
      ],
    },
  },

  'cli-commands': {
    id: 'cli-commands',
    title: 'CLI Reference',
    category: 'Getting Started',
    description: 'Full reference of all HydroGo commands: serve, migrate, create-admin, and seed-demo.',
    headings: ['Core Commands', 'Command Usage Examples'],
    content: {
      lead: 'The hydro command-line interface provides operational tools for migrations, administrative bootstrapping, and running the server.',
      sections: [
        {
          title: 'Core Commands',
          id: 'core-commands',
          text: 'HydroGo provides self-documenting commands for operational tasks.',
          table: {
            headers: ['Command', 'Description', 'Key Flags'],
            rows: [
              ['hydro serve', 'Starts the headless CMS HTTP server', '--config, --fastcgi'],
              ['hydro migrate', 'Runs all pending database schema migrations', '--config, --rollback-on-failure'],
              ['hydro create-admin', 'Creates or updates an administrator user', '--email, --password-stdin'],
              ['hydro seed-demo', 'Seeds disposable sample content for testing', '--confirm-demo, --locale'],
            ],
          },
        },
        {
          title: 'Command Usage Examples',
          id: 'command-usage-examples',
          text: 'Bootstrapping a brand new server in production with automated migrations and an initial administrator account:',
          code: {
            lang: 'bash',
            filename: 'terminal',
            code: `./bin/hydro migrate --config config.yaml
printf '%s' 'YourStrongPassword123!' | ./bin/hydro create-admin --email admin@example.com --password-stdin --config config.yaml
./bin/hydro serve --config config.yaml`,
          },
        },
      ],
    },
  },

  'content-types': {
    id: 'content-types',
    title: 'Declarative Content Types',
    category: 'Content & Modeling',
    badge: 'Zero Code',
    description: 'Create custom content structures with zero Go code and zero frontend recompilation.',
    headings: ['Defining Content Types', 'Schema Properties', 'Instant Admin & API'],
    content: {
      lead: 'Define structured collections like FAQs, Team Members, Portfolios, or Testimonials by dropping a single YAML file into content-types/.',
      sections: [
        {
          title: 'Defining Content Types',
          id: 'defining-content-types',
          text: 'HydroGo parses all *.yaml files in content-types/ at startup. It automatically generates the database schema, Admin Studio editing forms, and public read APIs.',
          code: {
            lang: 'yaml',
            filename: 'content-types/faq.yaml',
            code: `id: faq
name: FAQ
singular: Question
plural: Questions
public: true
list:
  search: [question, answer]
  default_sort: -created_at
fields:
  - id: question
    name: Question
    type: text
    required: true
    localized: true
  - id: answer
    name: Answer
    type: rich_text
    required: true
    localized: true
  - id: category
    name: Category
    type: select
    options: [General, Billing, Technical]`,
          },
          callout: {
            type: 'tip',
            text: 'Setting localized: true makes that field translatable across all configured locales (e.g. en, tr, de).',
          },
        },
        {
          title: 'Instant Admin & API',
          id: 'instant-admin-and-api',
          text: 'With the schema above, Admin Studio immediately renders a management UI at /admin/content/faq, and your frontend can query GET /api/v1/content/faq without writing a single line of backend code.',
        },
      ],
    },
  },

  'field-types': {
    id: 'field-types',
    title: 'Field Types & Validation',
    category: 'Content & Modeling',
    description: 'Comprehensive guide to all supported field kinds in HydroGo content types.',
    headings: ['Supported Kinds', 'Field Options'],
    content: {
      lead: 'Each field in your YAML schema accepts attributes controlling its presentation, validation, and localization.',
      sections: [
        {
          title: 'Supported Kinds',
          id: 'supported-kinds',
          text: 'HydroGo supports rich primitives designed for headless publishing.',
          table: {
            headers: ['Kind', 'JSON Type', 'Admin Control'],
            rows: [
              ['text', 'string', 'Single-line text input'],
              ['rich_text', 'string (HTML/Markdown)', 'TipTap / ProseMirror rich editor with blocks'],
              ['media', 'string (media UUID)', 'Media library picker with thumbnail preview'],
              ['select', 'string', 'Dropdown menu with configured options'],
              ['boolean', 'boolean', 'Switch toggle control'],
              ['datetime', 'string (ISO 8601)', 'Date & time picker'],
              ['relation', 'string (entity UUID)', 'Searchable relationship picker to another collection'],
            ],
          },
        },
      ],
    },
  },

  'pages-and-news': {
    id: 'pages-and-news',
    title: 'Pages & News Modules',
    category: 'Content & Modeling',
    description: 'Built-in CMS modules for hierarchical pages and news articles.',
    headings: ['Pages Module', 'News Module'],
    content: {
      lead: 'Pages and News are first-class core modules designed for rich editorial workflows.',
      sections: [
        {
          title: 'Pages Module',
          id: 'pages-module',
          text: 'Pages support hierarchical URLs, structured layout blocks, custom SEO metadata, and draft previews before publishing.',
        },
        {
          title: 'News Module',
          id: 'news-module',
          text: 'News articles feature scheduled publishing, tags, categories, author attribution, and automated OpenGraph image cards.',
        },
      ],
    },
  },

  'media-library': {
    id: 'media-library',
    title: 'Media Engine & WebP',
    category: 'Content & Modeling',
    description: 'Automatic WebP conversion, focal-point cropping, and BlurHash previews.',
    headings: ['Media Pipelines', 'Serving Assets'],
    content: {
      lead: 'HydroGo includes an automated media processing pipeline in pure Go without external C library dependencies.',
      sections: [
        {
          title: 'Media Pipelines',
          id: 'media-pipelines',
          text: 'When images are uploaded to Admin Studio, HydroGo automatically generates WebP variants, calculates focal point coordinates, and produces compact BlurHash placeholder strings for smooth UI loading.',
          callout: {
            type: 'note',
            text: 'Uploaded media is stored by SHA-256 hash, preventing duplicate file storage on disk or S3.',
          },
        },
      ],
    },
  },

  'api-overview': {
    id: 'api-overview',
    title: 'API Architecture & Envelope',
    category: 'REST API Reference',
    badge: 'REST',
    description: 'Learn about the unified JSON envelope, authentication tokens, and error formats.',
    headings: ['Standard Response Envelope', 'Error Format', 'Request ID Tracking'],
    content: {
      lead: 'Every public and administrative API endpoint returns JSON wrapped in an explicit envelope.',
      sections: [
        {
          title: 'Standard Response Envelope',
          id: 'standard-response-envelope',
          text: 'Successful endpoints always return { data, meta }:',
          code: {
            lang: 'json',
            filename: 'HTTP 200 OK',
            code: `{
  "data": [
    {
      "id": "item-101",
      "title": "Welcome to HydroGo",
      "slug": "welcome"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "limit": 20
  }
}`,
          },
        },
        {
          title: 'Error Format',
          id: 'error-format',
          text: 'Failed responses return { code, message, request_id, fields } with an HTTP error status:',
          code: {
            lang: 'json',
            filename: 'HTTP 404 / 400',
            code: `{
  "code": "not_found",
  "message": "The requested resource was not found",
  "request_id": "f8a1e204c3b5"
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
    description: 'Retrieve published pages and layout blocks for your frontend.',
    headings: ['List Pages', 'Get Page by Slug'],
    content: {
      lead: 'Query published site pages with full block layouts and localization.',
      sections: [
        {
          title: 'List Pages',
          id: 'list-pages',
          text: 'Returns an array of published pages for a given locale.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages?locale=en"',
          },
        },
        {
          title: 'Get Page by Slug',
          id: 'get-page-by-slug',
          text: 'Fetches a single page with all structured content blocks.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/pages/by-slug?locale=en&slug=about"',
          },
        },
      ],
    },
  },

  'news-api': {
    id: 'news-api',
    title: 'News & Articles Endpoints',
    category: 'REST API Reference',
    description: 'Fetch news articles, tags, authors, and cover media.',
    headings: ['List News', 'Get Article by Slug'],
    content: {
      lead: 'Retrieve news articles with pagination, tags, and category filtering.',
      sections: [
        {
          title: 'List News',
          id: 'list-news',
          text: 'Returns published articles ordered by publish timestamp.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/news?locale=en&limit=10"',
          },
        },
      ],
    },
  },

  'custom-content-api': {
    id: 'custom-content-api',
    title: 'Custom Content Endpoints',
    category: 'REST API Reference',
    description: 'Consume YAML-defined collections over the REST API.',
    headings: ['Querying Collections'],
    content: {
      lead: 'Any YAML schema with public: true gets a read endpoint at /api/v1/content/<id>.',
      sections: [
        {
          title: 'Querying Collections',
          id: 'querying-collections',
          text: 'Filter, search, and paginate through custom content items effortlessly.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl "http://127.0.0.1:8090/api/v1/content/faq?search=billing"',
          },
        },
      ],
    },
  },

  'system-probes': {
    id: 'system-probes',
    title: 'Health & Probes (/readyz)',
    category: 'REST API Reference',
    description: 'Readiness and liveness checks for load balancers and container orchestrators.',
    headings: ['Readiness Probe'],
    content: {
      lead: 'Health endpoints for Docker, Kubernetes, systemd, or Cloudflare probes.',
      sections: [
        {
          title: 'Readiness Probe',
          id: 'readiness-probe',
          text: 'Returns 200 OK ready when the database connection is healthy and migrations are verified.',
          code: {
            lang: 'bash',
            filename: 'cURL',
            code: 'curl -i http://127.0.0.1:8090/readyz',
          },
        },
      ],
    },
  },

  'admin-overview': {
    id: 'admin-overview',
    title: 'Svelte 5 Architecture',
    category: 'Admin Studio',
    description: 'How the embedded Svelte 5 SPA is structured, built, and served.',
    headings: ['Single Page Application', 'Zero-Latency Serving'],
    content: {
      lead: 'Admin Studio is an embedded Svelte 5 Single Page Application built with TypeScript, Tailwind CSS, and shadcn-svelte.',
      sections: [
        {
          title: 'Single Page Application',
          id: 'single-page-application',
          text: 'All admin views, components, and schema renderers live in admin/src/. The Vite build bundles them into internal/adminui/dist, which Go embeds directly into the compiled executable using go:embed.',
        },
        {
          title: 'Zero-Latency Serving',
          id: 'zero-latency-serving',
          text: 'Because admin assets are embedded inside Go memory with strong ETag headers, admin panel navigation is instant with zero disk I/O latency.',
        },
      ],
    },
  },

  'admin-rbac': {
    id: 'admin-rbac',
    title: 'Role-Based Access Control',
    category: 'Admin Studio',
    description: 'Permissions, user roles, and granular security overrides.',
    headings: ['User Roles', 'Permissions Model'],
    content: {
      lead: 'HydroGo enforces strict RBAC across all administrative routes.',
      sections: [
        {
          title: 'User Roles',
          id: 'user-roles',
          text: 'Supported roles: admin (full access), editor (content management), and viewer (read-only inspection).',
        },
      ],
    },
  },

  'admin-revisions': {
    id: 'admin-revisions',
    title: 'Drafts & Revisions',
    category: 'Admin Studio',
    description: 'Safe draft editing, version history, and rollbacks.',
    headings: ['Revision History'],
    content: {
      lead: 'Every publish action creates an immutable revision record allowing editors to rollback to earlier states safely.',
      sections: [
        {
          title: 'Revision History',
          id: 'revision-history',
          text: 'Revisions capture author ID, timestamp, and field deltas so changes can be audited and restored at any time.',
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
          text: 'Build the standalone binary with Go:',
          code: {
            lang: 'bash',
            filename: 'terminal',
            code: 'go build -ldflags="-s -w" -o bin/hydro ./cmd/hydro',
          },
        },
        {
          title: 'Systemd Service',
          id: 'systemd-service',
          text: 'Run HydroGo under systemd with automatic restart on failure and sandboxed file permissions.',
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
          text: 'Caddy provides automated Let\'s Encrypt certificates in two lines of configuration:',
          code: {
            lang: 'nginx',
            filename: 'Caddyfile',
            code: `cms.yourdomain.com {
  reverse_proxy 127.0.0.1:8090
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
      ],
    },
  },
};
