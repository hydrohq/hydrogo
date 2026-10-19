# Changelog

All notable changes to HydroGo are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-19

### Added
- **Single-Binary Headless CMS Core**: Core engine providing pages, news, media, members, site configuration, and collections over a clean REST API.
- **Embedded Svelte 5 Admin Panel**: Full SPA panel embedded in Go binary (`/admin`) powered by shadcn-svelte and TanStack Query.
- **Schema-Driven Content Types**: Declarative YAML schema engine (`content-types/`) generating API endpoints and admin UI automatically.
- **Multi-Dialect Database Engine**: Full support and automated migrations for SQLite, PostgreSQL (with RLS), and MySQL.
- **FastCGI & CGI Production Runtime**: Native `hydro serve --fastcgi` and `--cgi` modes for shared hosting and Apache/Nginx web servers.
- **Native View Analytics**: Atomic, privacy-preserving view counter (`POST /api/v1/news/views`, `GET /api/v1/news/views`) integrated with `analytics_content_daily`.
- **Social Crawler Interceptor**: Server-side metadata pre-rendering (`bot-meta.php`) serving OpenGraph, Twitter Cards, and NewsArticle JSON-LD to WhatsApp, Facebook, X (Twitter), LinkedIn, Telegram, Discord, and Google Inspection bots.
- **Self-Hosted Privacy Typography**: Self-hosted Public Sans fonts (`@fontsource/public-sans`) removing Google Fonts tracking.
- **Automated Deployment Pipeline**: Automated build and FTP deployment toolchain (`scripts/deploy-ardalabs.ps1`) supporting `-FrontendOnly` hot updates.

### Changed
- **Decoupled View Tracking**: Replaced fragile, file-based SvelteKit view counter (`views.json`) with persistent Go backend API endpoints.
- **Endpoint Hardening**: Added token authorization guards to `/api/admin/rss-ingest` and `/api/translate`.
- **TTS Abuse Prevention**: Protected `/api/tts` with sliding-window rate limiting (10 req/min) and 1,000 character payload limits.

### Deprecated
- **Legacy Commerce Peripherals**: Frozen legacy commerce tables and `internal/contracts/customerorders` capability contracts (ADR 0006).

### Security
- **Credential Sanitation**: Completely removed hardcoded fallback passwords and admin credentials across all starter scripts and seeds.
- **Secure Cookie & HSTS Presets**: Enforced strict transport and cookie security presets for production HTTPS environments.
