import type { AdminSchema, User } from '$lib/api/types';

export interface DemoState {
  authenticated: boolean;
  user: User;
  schema: AdminSchema;
  pages: Array<Record<string, any>>;
  news: Array<Record<string, any>>;
  content: Record<string, Array<Record<string, any>>>;
  users: Array<Record<string, any>>;
  members: Array<Record<string, any>>;
  media: Array<Record<string, any>>;
  audit: Array<Record<string, any>>;
  notifications: Array<Record<string, any>>;
  site: Record<string, any>;
}

function daysAgoIso(days: number): string {
  const d = new Date(Date.now() - days * 86400_000);
  return d.toISOString();
}

function buildDailyAnalytics(): Array<{ Day: string; Metric: string; Dimension: string; EventCount: number }> {
  const out: Array<{ Day: string; Metric: string; Dimension: string; EventCount: number }> = [];
  for (let i = 29; i >= 0; i--) {
    const d = new Date(Date.now() - i * 86400_000);
    const day = d.toISOString().slice(0, 10);
    const base = 240 + ((i * 43) % 180);
    out.push({ Day: day, Metric: 'page_view', Dimension: 'desktop', EventCount: base });
    out.push({ Day: day, Metric: 'page_view', Dimension: 'mobile', EventCount: Math.round(base * 0.68) });
  }
  return out;
}

export function createInitialDemoState(): DemoState {
  const adminUser: User = {
    id: 'usr_demo_01',
    email: 'admin@example.com',
    role: 'admin',
    active: true,
    mfa_enabled: false,
  };

  return {
    authenticated: false,
    user: adminUser,
    schema: {
      version: 1,
      platform: false,
      site_url: 'https://hydrogo.dev',
      locales: { enabled: true, default: 'en', all: ['en', 'tr'] },
      content_types: [],
      permissions: ['*'],
      modules: {
        pages: true,
        news: true,
        members: true,
        commerce: false,
        ai: true,
      },
      resources: [
        {
          id: 'pages',
          label: 'Pages',
          singular: 'Page',
          source: 'builtin',
          type: 'collection',
          permissions: { view: 'content:read', create: 'content:write', edit: 'content:write', delete: 'content:write' },
          fields: [
            { name: 'title', label: 'Title', kind: 'text', required: true },
            { name: 'slug', label: 'Slug', kind: 'text', required: true },
            { name: 'status', label: 'Status', kind: 'select', options: ['draft', 'published'] },
          ],
        },
        {
          id: 'news',
          label: 'News',
          singular: 'Post',
          source: 'builtin',
          type: 'collection',
          permissions: { view: 'content:read', create: 'content:write', edit: 'content:write', delete: 'content:write' },
          fields: [
            { name: 'title', label: 'Title', kind: 'text', required: true },
            { name: 'slug', label: 'Slug', kind: 'text', required: true },
            { name: 'category', label: 'Category', kind: 'text' },
            { name: 'status', label: 'Status', kind: 'select', options: ['draft', 'published'] },
          ],
        },
        {
          id: 'changelog',
          label: 'Changelog',
          singular: 'Release',
          source: 'content_type',
          type: 'collection',
          permissions: { view: 'content:read', create: 'content:write', edit: 'content:write', delete: 'content:write' },
          fields: [
            { name: 'version', label: 'Version', kind: 'text', required: true },
            { name: 'summary', label: 'Summary', kind: 'textarea', required: true },
            { name: 'status', label: 'Status', kind: 'select', options: ['draft', 'published'] },
          ],
        },
        {
          id: 'testimonials',
          label: 'Testimonials',
          singular: 'Testimonial',
          source: 'content_type',
          type: 'collection',
          permissions: { view: 'content:read', create: 'content:write', edit: 'content:write', delete: 'content:write' },
          fields: [
            { name: 'author', label: 'Source', kind: 'text', required: true },
            { name: 'role', label: 'Team', kind: 'text' },
            { name: 'quote', label: 'Quote', kind: 'textarea', required: true },
          ],
        },
      ],
    },
    pages: [
      {
        id: 'pg_home',
        title: 'Landing Page',
        slug: 'home',
        status: 'published',
        author_name: 'admin@example.com',
        body: 'Single-binary Go backend engine for applications, content, and high-scale APIs.',
        updated_at: daysAgoIso(0),
        created_at: daysAgoIso(30),
      },
      {
        id: 'pg_arch',
        title: 'System Architecture',
        slug: 'architecture',
        status: 'published',
        author_name: 'admin@example.com',
        body: 'Multi-dialect SQL storage with SQLite WAL, PostgreSQL RLS, and MySQL support.',
        updated_at: daysAgoIso(1),
        created_at: daysAgoIso(25),
      },
      {
        id: 'pg_security',
        title: 'Security & Row-Level Isolation',
        slug: 'security-rls',
        status: 'published',
        author_name: 'editor@example.com',
        body: 'CSRF double-submit tokens, magic-byte upload verification, and tenant-scoped queries.',
        updated_at: daysAgoIso(2),
        created_at: daysAgoIso(18),
      },
      {
        id: 'pg_api',
        title: 'REST & MCP API Reference',
        slug: 'api-reference',
        status: 'published',
        author_name: 'editor@example.com',
        body: 'Auto-generated REST endpoints and native Model Context Protocol server with per-type permissions.',
        updated_at: daysAgoIso(4),
        created_at: daysAgoIso(15),
      },
      {
        id: 'pg_deploy',
        title: 'Production Deployment Guide',
        slug: 'deployment',
        status: 'published',
        author_name: 'ops@example.com',
        body: 'Single-binary systemd service behind Caddy or Cloudflare with automated SQLite/Postgres backups.',
        updated_at: daysAgoIso(6),
        created_at: daysAgoIso(12),
      },
      {
        id: 'pg_roadmap',
        title: 'Q4 Engineering Roadmap',
        slug: 'roadmap',
        status: 'draft',
        author_name: 'admin@example.com',
        body: 'Upcoming improvements for distributed caching and automated schema migrations.',
        updated_at: daysAgoIso(0),
        created_at: daysAgoIso(5),
      },
    ],
    news: [
      {
        id: 'nw_051',
        title: 'HydroGo v0.5.1 Security & Studio Polish',
        slug: 'hydrogo-v051-release',
        category: 'Release',
        status: 'published',
        featured: true,
        author_name: 'admin@example.com',
        summary: 'Hardened upload validation, MCP per-type permissions, and embedded Svelte 5 studio refinements.',
        body: 'v0.5.1 delivers end-to-end quarantine upload checks, strict tenant isolation, and a cleaner Admin Studio.',
        published_at: daysAgoIso(0),
        created_at: daysAgoIso(1),
        updated_at: daysAgoIso(0),
      },
      {
        id: 'nw_rls',
        title: 'Row-Level Security across SQLite & PostgreSQL',
        slug: 'rls-across-dialects',
        category: 'Security',
        status: 'published',
        featured: false,
        author_name: 'editor@example.com',
        summary: 'How HydroGo enforces tenant boundaries uniformly at the database and query layer.',
        body: 'Every scoped query binds tenant context automatically before hitting SQLite or PostgreSQL.',
        published_at: daysAgoIso(3),
        created_at: daysAgoIso(4),
        updated_at: daysAgoIso(3),
      },
      {
        id: 'nw_mcp',
        title: 'Per-Type Permissions in the Native MCP Server',
        slug: 'mcp-per-type-permissions',
        category: 'Engineering',
        status: 'published',
        featured: false,
        author_name: 'admin@example.com',
        summary: 'Enforcing schema-level read and write rules for AI agents and external tool calls.',
        body: 'MCP query_collection and mutate_record tools now evaluate collection permissions identically to HTTP handlers.',
        published_at: daysAgoIso(5),
        created_at: daysAgoIso(6),
        updated_at: daysAgoIso(5),
      },
      {
        id: 'nw_studio',
        title: 'Embedded Svelte 5 Admin Studio Benchmarks',
        slug: 'svelte5-studio-benchmarks',
        category: 'Architecture',
        status: 'published',
        featured: false,
        author_name: 'ops@example.com',
        summary: 'Zero-dependency single-binary embedding with sub-millisecond cold start.',
        body: 'All admin assets are compiled and embedded directly into the Go binary via embed.FS.',
        published_at: daysAgoIso(8),
        created_at: daysAgoIso(9),
        updated_at: daysAgoIso(8),
      },
      {
        id: 'nw_draft',
        title: 'Zero-Downtime Backup & Restore Drills',
        slug: 'backup-restore-drills',
        category: 'Engineering',
        status: 'draft',
        featured: false,
        author_name: 'ops@example.com',
        summary: 'Automated snapshot verification for SQLite WAL and PostgreSQL.',
        body: 'Draft runbook for verifying backup integrity before every release.',
        published_at: daysAgoIso(1),
        created_at: daysAgoIso(2),
        updated_at: daysAgoIso(1),
      },
    ],
    content: {
      changelog: [
        {
          id: 'chg_051',
          version: 'v0.5.1',
          summary: 'Direct upload magic-byte checks, MCP per-type permissions, and CSRF shape validation.',
          status: 'published',
          updated_at: daysAgoIso(0),
        },
        {
          id: 'chg_050',
          version: 'v0.5.0',
          summary: 'Config extensions for add-on modules, commerce batch stock endpoint, and RLS default scope fix.',
          status: 'published',
          updated_at: daysAgoIso(4),
        },
        {
          id: 'chg_041',
          version: 'v0.4.1',
          summary: 'Container build fix, macOS symlink-free TMPDIR check runner, and dependency security updates.',
          status: 'published',
          updated_at: daysAgoIso(13),
        },
      ],
      testimonials: [
        {
          id: 'tst_01',
          author: 'Core Platform Team',
          role: 'Infrastructure',
          quote: 'Deploying a single static Go binary with embedded admin and migrations cut our ops overhead in half.',
          updated_at: daysAgoIso(10),
        },
        {
          id: 'tst_02',
          author: 'Storefront Engineering',
          role: 'Web & API',
          quote: 'Having SQLite locally and PostgreSQL RLS in production with the exact same schema contract is effortless.',
          updated_at: daysAgoIso(6),
        },
      ],
    },
    users: [
      { ...adminUser, name: 'System Admin', created_at: daysAgoIso(45), updated_at: daysAgoIso(1) },
      {
        id: 'usr_editor_02',
        email: 'editor@example.com',
        name: 'Editorial Service',
        role: 'editor',
        active: true,
        mfa_enabled: true,
        created_at: daysAgoIso(20),
        updated_at: daysAgoIso(2),
      },
      {
        id: 'usr_ops_03',
        email: 'ops@example.com',
        name: 'Operations Lead',
        role: 'editor',
        active: true,
        mfa_enabled: true,
        created_at: daysAgoIso(16),
        updated_at: daysAgoIso(3),
      },
      {
        id: 'usr_viewer_04',
        email: 'auditor@example.com',
        name: 'Security Auditor',
        role: 'viewer',
        active: true,
        mfa_enabled: false,
        created_at: daysAgoIso(12),
        updated_at: daysAgoIso(5),
      },
    ],
    members: [
      { id: 'mem_01', email: 'member-01@example.com', name: 'Workspace Account 01', active: true, email_verified_at: daysAgoIso(20), status: 'active', email_verified: true, created_at: daysAgoIso(21) },
      { id: 'mem_02', email: 'member-02@example.com', name: 'Workspace Account 02', active: true, email_verified_at: daysAgoIso(13), status: 'active', email_verified: true, created_at: daysAgoIso(14) },
      { id: 'mem_03', email: 'member-03@example.com', name: 'API Consumer 03', active: true, email_verified_at: daysAgoIso(8), status: 'active', email_verified: true, created_at: daysAgoIso(9) },
      { id: 'mem_04', email: 'member-04@example.com', name: 'Partner Client 04', active: true, email_verified_at: null, status: 'active', email_verified: false, created_at: daysAgoIso(5) },
      { id: 'mem_05', email: 'member-05@example.com', name: 'Webhook Client 05', active: true, email_verified_at: daysAgoIso(2), status: 'active', email_verified: true, created_at: daysAgoIso(3) },
      { id: 'mem_06', email: 'member-06@example.com', name: 'Sandbox Account 06', active: false, email_verified_at: null, status: 'pending', email_verified: false, created_at: daysAgoIso(1) },
    ],
    media: [
      {
        id: 'med_01',
        original_filename: 'hydrogo-gopher.svg',
        mime_type: 'image/svg+xml',
        size_bytes: 18420,
        url: '/images/gophers/burton.svg',
        created_at: daysAgoIso(12),
      },
      {
        id: 'med_02',
        original_filename: 'release-notes-v051.pdf',
        mime_type: 'application/pdf',
        size_bytes: 245100,
        url: '#demo-file',
        created_at: daysAgoIso(1),
      },
      {
        id: 'med_03',
        original_filename: 'security-checklist.json',
        mime_type: 'application/json',
        size_bytes: 12840,
        url: '#demo-file',
        created_at: daysAgoIso(2),
      },
    ],
    audit: [
      { id: 'aud_01', at: daysAgoIso(0), action: 'auth.login', resource: 'session', actor: 'admin@example.com', target: 'session', request_id: 'req_9f8a12bc34de', after_json: JSON.stringify({ ip: '127.0.0.1', method: 'POST', email: 'admin@example.com' }) },
      { id: 'aud_02', at: daysAgoIso(0), action: 'news.publish', resource: 'news', actor: 'admin@example.com', target: 'nw_051', request_id: 'req_7c41de90ab12', after_json: JSON.stringify({ ip: '127.0.0.1', method: 'PUT', email: 'admin@example.com' }) },
      { id: 'aud_03', at: daysAgoIso(1), action: 'security.upload_quarantine_ok', resource: 'media', actor: 'editor@example.com', target: 'med_02', request_id: 'req_5b2190ff11aa', after_json: JSON.stringify({ ip: '10.0.0.14', method: 'POST', email: 'editor@example.com' }) },
      { id: 'aud_04', at: daysAgoIso(2), action: 'pages.update', resource: 'pages', actor: 'editor@example.com', target: 'pg_security', request_id: 'req_4a1288ef22bb', after_json: JSON.stringify({ ip: '10.0.0.14', method: 'PATCH', email: 'editor@example.com' }) },
      { id: 'aud_05', at: daysAgoIso(3), action: 'news.create', resource: 'news', actor: 'ops@example.com', target: 'nw_rls', request_id: 'req_2c9011aa77cc', after_json: JSON.stringify({ ip: '10.0.0.22', method: 'POST', email: 'ops@example.com' }) },
      { id: 'aud_06', at: daysAgoIso(4), action: 'schema.update', resource: 'content_types', actor: 'admin@example.com', target: 'changelog', request_id: 'req_3e1099cc8877', after_json: JSON.stringify({ ip: '127.0.0.1', method: 'PATCH', email: 'admin@example.com' }) },
    ],
    notifications: [
      { id: 'ntf_01', title: 'v0.5.1 Live Demo Active', message: 'Changes are stored locally in your browser localStorage.', read: false, created_at: daysAgoIso(0) },
      { id: 'ntf_02', title: 'Security Check Passed', message: 'All 62 static routes and backend checks passed.', read: true, created_at: daysAgoIso(1) },
    ],
    site: {
      general: {
        site_name: 'HydroGo Demo Workspace',
        site_url: 'https://hydrogo.dev',
        default_locale: 'en',
        tagline: 'Single-binary Go backend engine',
      },
      maintenance: {
        enabled: false,
        headline: 'Scheduled Maintenance',
        message: 'We will be back shortly.',
        allowlist: ['127.0.0.1'],
      },
      notice: {
        active: false,
        message: '',
        variant: 'info',
      },
    },
  };
}

export function buildDashboardPayload(state: DemoState) {
  return {
    content: {
      pages_published: state.pages.filter((p) => p.status === 'published').length,
      news_published: state.news.filter((n) => n.status === 'published').length,
    },
    members: {
      total_members: state.members.length,
      active_members: state.members.filter((m) => m.status === 'active').length,
      verified_members: state.members.filter((m) => m.email_verified).length,
    },
    runtime: {
      uptime_seconds: 172800,
      requests: 28450,
      views: 18920,
      errors: 1,
      avg_latency_ms: 3.8,
    },
    analytics: {
      unique_visitors_today: 612,
      daily: buildDailyAnalytics(),
    },
  };
}
