import { type DocSection, page, p, h2, h3, code, note, tip, warn, table, ul } from './types';

export const configuration: DocSection = {
  id: 'configuration',
  title: 'Configuration',
  pages: [
    page(
      'config-file',
      'config.yaml reference',
      "Every commonly used key in config.yaml, what it does and its default. Copy config.example.yaml from the repository for a commented starting point.",
      p("HydroGo reads `config.yaml` (path given with `--config`). Every key can also be overridden with a `HYDRO_*` environment variable, listed on [Environment variables](doc:configuration/environment-variables). Environment variables win over the file, and command-line runtime flags win over both."),
      code('yaml', `profile: compact
modules: [core, auth, members, pages, media, site, news, collections]
http:
  address: "127.0.0.1:8080"
  public_origin: "https://cms.example.com"
database:
  dialect: "sqlite"
  dsn: "data/hydro.db"
media:
  dir: "data/media"
  min_free_bytes: 52428800
i18n:
  enabled: true
  default_locale: en
  locales: [en, tr]
security:
  secure_cookies: true
  enable_hsts: true
audit:
  log_request_bodies: false`, 'config.yaml'),
      tip("Validate a file before you start the server: `hydro validate --config config.yaml`. It reports configuration and database problems with a remediation hint."),
      h2('Top level'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`profile`', '`compact`', 'Capacity profile: `compact`, `standard` or `distributed`. See [Modules and profiles](doc:configuration/modules-and-profiles).'],
        ['`modules`', 'per preset', 'Which modules run: `core`, `auth`, `members`, `pages`, `media`, `site`, `news`, `collections`, `analytics`.'],
        ['`content_types_dir`', '`content-types`', 'Folder with the content type YAML files.'],
        ['`safe_mode`', '`false`', 'Start with a restricted feature set for recovery.'],
      ),
      h2('http'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`address`', '`127.0.0.1:8080`', 'Listen address. Use `0.0.0.0:8080` in containers.'],
        ['`public_origin`', 'empty', 'Canonical public HTTPS origin of the CMS, for example `https://cms.example.com`. Set it in every deployed environment.'],
        ['`behind_proxy`', '`false`', 'Trust that a reverse proxy terminates TLS in front of HydroGo.'],
        ['`auto_tls`', '`false`', 'Obtain certificates automatically. Needs `domain` and cannot be combined with `behind_proxy`.'],
        ['`domain`', 'empty', 'The single domain used for automatic TLS.'],
      ),
      h2('database'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`dialect`', '`sqlite`', '`sqlite`, `postgres` or `mysql` (MySQL and MariaDB).'],
        ['`dsn`', 'empty', 'Connection string. Prefer `HYDRO_DATABASE_DSN` or a protected secret file over putting credentials in the YAML.'],
        ['`sqlite_external_write_watch`', '`false`', 'Detect external writers to the SQLite file.'],
      ),
      code('text', `# PostgreSQL
postgres://hydro_app:REPLACE_ME@db.internal:5432/hydro?sslmode=verify-full
# MySQL / MariaDB
hydro:REPLACE_ME@tcp(db.internal:3306)/hydro
# SQLite (a file path)
data/hydro.db`, 'DSN examples'),
      h2('media'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`dir`', '`data/media`', 'Local media directory.'],
        ['`min_free_bytes`', '`52428800`', 'Refuse uploads when free disk space drops below this (50 MiB).'],
        ['`storage_driver`', 'local', '`s3` stores media in an S3-compatible bucket.'],
        ['`s3_endpoint`, `s3_bucket`, `s3_region`', 'empty', 'S3 location.'],
        ['`s3_access_key`, `s3_secret_key`', 'empty', 'S3 credentials. Use environment variables.'],
        ['`s3_use_ssl`, `s3_force_path_style`', '', 'S3 connection options (path-style is needed by many self-hosted stores).'],
        ['`cdn_base_url`', 'empty', 'Serve media URLs from a CDN origin.'],
        ['`hotlink_protection`, `allowed_referer_hosts`', 'off', 'Block other sites from embedding your media.'],
        ['`watermark_text`, `watermark_position`, `watermark_opacity`, `watermark_margin`', 'off', 'Optional text watermark on image derivatives.'],
      ),
      h2('i18n'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`enabled`', '', 'Turn on content locales.'],
        ['`default_locale`', '`en`', 'Fallback locale. Cannot be empty.'],
        ['`locales`', '', 'List of 1 to 64 locale codes such as `en`, `tr`, `pt-BR`.'],
      ),
      p("Content locales are covered in [Content locales](doc:multi-language/content-locales)."),
      h2('security'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`secure_cookies`', '`false`', 'Mark cookies `Secure`. Keep `false` for plain-HTTP local development, `true` on HTTPS.'],
        ['`enable_hsts`', '`false`', 'Send `Strict-Transport-Security`. Enable only when HTTPS works everywhere.'],
        ['`headers_preset`', '`strict`', 'Preset of security response headers.'],
        ['`enable_anomaly_filter`', '`true`', 'Block obviously abusive requests.'],
        ['`anomaly_bypass_paths`', '`[]`', 'Paths exempt from the anomaly filter.'],
        ['`mfa_required_roles`', '`[]`', 'Roles that must use MFA, for example `[admin]`. Enrol the accounts first.'],
        ['`mfa_master_key`', 'empty', 'Secret (16, 24 or 32 bytes) that protects stored TOTP seeds.'],
        ['`public_api_origins`', '`[]`', 'Explicit allowlist of browser origins that may read published `/api/v1` content without credentials. Wildcards are not supported.'],
        ['`preview_signing_key`', 'empty', 'At least 32 bytes; signs draft preview links.'],
        ['`password_reset_ttl`, `invitation_ttl`', 'built-in', 'Lifetime of reset and invitation links, for example `1h`.'],
      ),
      h2('audit'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`log_request_bodies`', '`false`', 'Record the redacted, size-capped JSON body of every state-changing admin API request in the audit trail. See [Audit trail](doc:security/audit-trail).'],
      ),
      h2('redis'),
      table(
        ['Key', 'Default', 'Meaning'],
        ['`url`', 'empty', 'Redis or Valkey URL, e.g. `rediss://:secret@host:6379/0`. Leave empty for single-instance setups.'],
        ['`key_prefix`', '`hydro:`', 'Prefix for all keys, so several apps can share one Redis.'],
      ),
      h2('mail and forms'),
      table(
        ['Key', 'Meaning'],
        ['`mail.host`, `mail.port`, `mail.username`, `mail.password`, `mail.from`, `mail.starttls`', 'SMTP server for transactional e-mail (invitations, password resets, notifications).'],
        ['`mail.dkim_selector`', 'DKIM selector for signed mail.'],
        ['`mail.max_per_second`, `mail.max_per_hour`', 'Outgoing mail rate limits.'],
        ['`forms.turnstile_site_key`, `forms.turnstile_secret_key`', 'Cloudflare Turnstile keys for public forms.'],
      ),
      h2('seo, analytics, geoip, observability'),
      table(
        ['Key', 'Meaning'],
        ['`seo.og_cache_dir`, `seo.fonts_dir`', 'Cache and font folders for generated Open Graph images (`/og/{locale}/{slug}`).'],
        ['`analytics.no_cookie_enabled`, `analytics.hash_key`', 'Opt-in cookie-free daily visitor summaries; the hash key must be 32+ random bytes.'],
        ['`geoip.database_path`, `geoip.trusted_proxy_cidrs`', 'Optional offline GeoIP database and the proxy ranges that may supply location headers.'],
        ['`observability.sentry_dsn`', 'Sentry or GlitchTip DSN for error reporting.'],
      ),
      h2('webhooks and backup'),
      p("`webhooks` is a list of outgoing endpoints; see [Caching and webhooks](doc:frontends/caching-and-webhooks). Scheduled cloud backups use the `backup` block."),
      code('yaml', `webhooks:
  - id: frontend-rebuild
    url: https://ci.example.com/hooks/rebuild
    topics: [page.published, news.published, news.retracted]
backup:
  enabled: false
  prefix: backups/
  retention_days: 30
  hour_utc: 3`),
      note("`backup.enabled: true` requires configured S3-compatible credentials, a `retention_days` between 1 and 3650 and an `hour_utc` between 0 and 23. Otherwise the config fails validation."),
      h2('Other blocks'),
      ul(
        "`tenancy` (`enabled`, `require_host`, `hosts`) maps hostnames to tenants for multi-tenant deployments.",
        "`admin_support` shows an optional support contact card inside the admin (`agency_name`, `email`, `phone`, `whatsapp_url`, `sla_badge`, `guide_url`).",
        "`admin_extensions` (`manifest_dir`, `trusted_keys_file`) loads signed, externally hosted admin surfaces.",
        "`license` (`required`, `file`, `project_id`, `edition`, `features`) is only relevant to licensed editions.",
      ),
    ),

    page(
      'environment-variables',
      'Environment variables',
      "Every HYDRO_* variable that overrides config.yaml, grouped by area. Use them for secrets and per-environment values.",
      p("Environment variables override the matching `config.yaml` key. Boolean values accept `true` and `false`; lists are comma separated. Invalid values stop the start-up with a message naming the variable."),
      warn("Keep secrets out of `config.yaml` and out of version control. Pass them as environment variables, through your orchestrator's secret store, or from a protected file."),
      h2('Core'),
      table(
        ['Variable', 'Overrides'],
        ['`HYDRO_HTTP_ADDRESS`', '`http.address`'],
        ['`HYDRO_PUBLIC_ORIGIN`', '`http.public_origin`'],
        ['`HYDRO_CAPACITY_PROFILE`', '`profile`'],
        ['`HYDRO_SAFE_MODE`', '`safe_mode`'],
        ['`HYDRO_CONTENT_TYPES_DIR`', '`content_types_dir`'],
        ['`HYDRO_DATABASE_DIALECT`', '`database.dialect`'],
        ['`HYDRO_DATABASE_DSN`', '`database.dsn`'],
        ['`HYDRO_SQLITE_EXTERNAL_WRITE_WATCH`', '`database.sqlite_external_write_watch`'],
        ['`HYDRO_REDIS_URL`, `HYDRO_REDIS_KEY_PREFIX`', '`redis.url`, `redis.key_prefix`'],
        ['`HYDRO_ADMIN_PASSWORD`', 'Password for the admin user created by `hydro new` or `hydro init`.'],
      ),
      h2('Security'),
      table(
        ['Variable', 'Overrides'],
        ['`HYDRO_SECURE_COOKIES`', '`security.secure_cookies`'],
        ['`HYDRO_ENABLE_HSTS`', '`security.enable_hsts`'],
        ['`HYDRO_PUBLIC_API_ORIGINS`', '`security.public_api_origins`'],
        ['`HYDRO_MFA_MASTER_KEY`', '`security.mfa_master_key`'],
        ['`HYDRO_PREVIEW_SIGNING_KEY`', '`security.preview_signing_key`'],
        ['`HYDRO_PASSWORD_RESET_TTL`, `HYDRO_INVITATION_TTL`', '`security.password_reset_ttl`, `security.invitation_ttl`'],
        ['`HYDRO_AUDIT_LOG_REQUEST_BODIES`', '`audit.log_request_bodies`'],
      ),
      h2('Media'),
      table(
        ['Variable', 'Overrides'],
        ['`HYDRO_MEDIA_DIR`, `HYDRO_MEDIA_MIN_FREE_BYTES`', '`media.dir`, `media.min_free_bytes`'],
        ['`HYDRO_MEDIA_STORAGE_DRIVER`', '`media.storage_driver` (`s3` for object storage)'],
        ['`HYDRO_MEDIA_S3_ENDPOINT`, `_BUCKET`, `_REGION`, `_ACCESS_KEY`, `_SECRET_KEY`', 'The `media.s3_*` keys (prefix each with `HYDRO_MEDIA_S3`)'],
        ['`HYDRO_MEDIA_S3_USE_SSL`, `HYDRO_MEDIA_S3_FORCE_PATH_STYLE`', '`media.s3_use_ssl`, `media.s3_force_path_style`'],
        ['`HYDRO_MEDIA_CDN_BASE_URL`', '`media.cdn_base_url`'],
        ['`HYDRO_MEDIA_HOTLINK_PROTECTION`, `HYDRO_MEDIA_ALLOWED_REFERER_HOSTS`', 'Hotlink protection and its allowed hosts'],
        ['`HYDRO_MEDIA_WATERMARK_TEXT`, `_POSITION`, `_OPACITY`, `_MARGIN`', 'Watermark settings (prefix `HYDRO_MEDIA_WATERMARK`)'],
      ),
      h2('Languages'),
      table(
        ['Variable', 'Overrides'],
        ['`HYDRO_I18N_ENABLED`', '`i18n.enabled`'],
        ['`HYDRO_DEFAULT_LOCALE`', '`i18n.default_locale`'],
        ['`HYDRO_LOCALES`', '`i18n.locales`, comma separated, for example `en,tr`'],
      ),
      h2('Mail, forms and integrations'),
      table(
        ['Variable', 'Overrides'],
        ['`HYDRO_SMTP_HOST`, `_PORT`, `_USERNAME`, `_PASSWORD`, `_FROM`, `_STARTTLS`, `_DKIM_SELECTOR`', 'The `mail.*` keys (prefix `HYDRO_SMTP`)'],
        ['`HYDRO_SMTP_MAX_PER_SECOND`, `HYDRO_SMTP_MAX_PER_HOUR`', '`mail.max_per_second`, `mail.max_per_hour`'],
        ['`HYDRO_FORMS_TURNSTILE_SITE_KEY`, `HYDRO_FORMS_TURNSTILE_SECRET_KEY`', '`forms.turnstile_*`'],
        ['`HYDRO_WEBHOOK_URL`, `HYDRO_WEBHOOK_SECRET`, `HYDRO_WEBHOOK_TOPICS`', 'A single webhook endpoint without editing YAML'],
        ['`HYDRO_SEO_OG_CACHE_DIR`, `HYDRO_SEO_FONTS_DIR`', '`seo.og_cache_dir`, `seo.fonts_dir`'],
        ['`HYDRO_ANALYTICS_NO_COOKIE_ENABLED`, `HYDRO_ANALYTICS_HASH_KEY`', '`analytics.*`'],
        ['`HYDRO_SENTRY_DSN`', '`observability.sentry_dsn`'],
        ['`HYDRO_BACKUP_SIGNING_SEED`', 'Hex seed (64 characters) that signs backup manifests'],
      ),
      h2('Example: production environment file'),
      code('env', `HYDRO_CAPACITY_PROFILE=standard
HYDRO_HTTP_ADDRESS=0.0.0.0:8080
HYDRO_PUBLIC_ORIGIN=https://cms.example.com
HYDRO_SECURE_COOKIES=true
HYDRO_ENABLE_HSTS=true
HYDRO_DATABASE_DIALECT=postgres
HYDRO_DATABASE_DSN=postgres://hydro_app:REPLACE_ME@db.internal:5432/hydro?sslmode=verify-full
HYDRO_REDIS_URL=rediss://:REPLACE_ME@redis.internal:6379/0
HYDRO_MFA_MASTER_KEY=REPLACE_WITH_16_24_OR_32_BYTE_SECRET
HYDRO_MEDIA_STORAGE_DRIVER=s3
HYDRO_MEDIA_S3_ENDPOINT=s3.eu-central-1.amazonaws.com
HYDRO_MEDIA_S3_BUCKET=example-media
HYDRO_MEDIA_S3_REGION=eu-central-1
HYDRO_MEDIA_CDN_BASE_URL=https://cdn.example.com
HYDRO_SMTP_HOST=smtp.example.com
HYDRO_SMTP_PORT=587
HYDRO_SMTP_FROM=noreply@example.com`, '.env'),
      tip("The repository ships `deploy/env.production.example`, `env.staging.example` and `env.dev.example` as templates."),
    ),

    page(
      'modules-and-profiles',
      'Modules and profiles',
      "Turn features on and off with the modules list, and choose a capacity profile that matches your deployment size.",
      h2('Modules'),
      p("A module is a unit of functionality with its own routes, migrations, workers and admin navigation. An inactive module adds none of those. Enable what you need in `modules`, then run `hydro migrate`."),
      table(
        ['Module', 'Provides'],
        ['`core`', 'HTTP server, config, health endpoints. Always on.'],
        ['`auth`', 'Admin users, sessions, roles, MFA, API keys and tokens.'],
        ['`members`', 'Visitor accounts and the member token API.'],
        ['`pages`', 'Pages with blocks, revisions and preview links.'],
        ['`media`', 'Media library and delivery.'],
        ['`site`', 'Site settings, navigation, redirects, forms, sitemap and maintenance mode.'],
        ['`news`', 'News articles, categories, tags and scheduled publishing.'],
        ['`collections`', 'The engine behind YAML content types.'],
        ['`analytics`', 'Opt-in, privacy-friendly traffic summaries.'],
      ),
      h2('Capacity profiles'),
      p("A profile sets sensible defaults for how much machinery runs inside the process. Providers such as the database, S3 and Redis are chosen separately; the profile does not infer them."),
      table(
        ['Profile', 'Use for', 'Adds'],
        ['`compact`', 'Single small site, local development, shared hosting', 'In-memory caches, minimal background work.'],
        ['`standard`', 'A production site on a VPS or container', 'Bounded background workers, optional distributed cache.'],
        ['`distributed`', 'Several instances behind a load balancer', 'Everything in `standard`, plus tenant admission and OpenTelemetry export (opt-in).'],
      ),
      code('bash', `hydro serve --config config.yaml --capacity-profile standard
# or
HYDRO_CAPACITY_PROFILE=standard hydro serve --config config.yaml`),
      note("Running more than one instance? Use PostgreSQL or MySQL and set `redis.url`, so rate limits and caches are shared."),
    ),
  ],
};
