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
    const base = 180 + ((i * 37) % 140);
    out.push({ Day: day, Metric: 'page_view', Dimension: 'desktop', EventCount: base });
    out.push({ Day: day, Metric: 'page_view', Dimension: 'mobile', EventCount: Math.round(base * 0.65) });
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
    authenticated: true,
    user: adminUser,
    schema: {
      version: 1,
      platform: false,
      site_url: 'https://hydrogo.pages.dev',
      locales: { enabled: true, default: 'en', all: ['en', 'tr'] },
      content_types: [],
      permissions: ['*'],
      modules: {
        commerce: false,
        members: true,
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
            { name: 'author', label: 'Author', kind: 'text', required: true },
            { name: 'role', label: 'Role', kind: 'text' },
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
        body: 'Single-binary Go backend engine for applications, content, and high-scale APIs.',
        updated_at: daysAgoIso(1),
        created_at: daysAgoIso(30),
      },
      {
        id: 'pg_arch',
        title: 'System Architecture',
        slug: 'architecture',
        status: 'published',
        body: 'Multi-dialect SQL storage with SQLite WAL, PostgreSQL RLS, and MySQL support.',
        updated_at: daysAgoIso(3),
        created_at: daysAgoIso(25),
      },
      {
        id: 'pg_roadmap',
        title: 'Q4 Engineering Roadmap',
        slug: 'roadmap',
        status: 'draft',
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
        status: 'published',
        featured: true,
        summary: 'Hardened upload validation, MCP per-type permissions, and embedded Svelte 5 studio refinements.',
        body: 'v0.5.1 delivers end-to-end quarantine upload checks, strict tenant isolation, and a cleaner Admin Studio.',
        published_at: daysAgoIso(0),
        updated_at: daysAgoIso(0),
      },
      {
        id: 'nw_rls',
        title: 'Row-Level Security across SQLite & PostgreSQL',
        slug: 'rls-across-dialects',
        status: 'published',
        featured: false,
        summary: 'How HydroGo enforces tenant boundaries uniformly at the database and query layer.',
        body: 'Every scoped query binds tenant context automatically before hitting SQLite or PostgreSQL.',
        published_at: daysAgoIso(7),
        updated_at: daysAgoIso(7),
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
      ],
      testimonials: [
        {
          id: 'tst_01',
          author: 'Deniz K.',
          role: 'Lead Platform Engineer',
          quote: 'Deploying a single static Go binary with embedded admin and migrations cut our ops overhead in half.',
          updated_at: daysAgoIso(10),
        },
      ],
    },
    users: [
      adminUser,
      {
        id: 'usr_editor_02',
        email: 'editor@example.com',
        name: 'Content Editor',
        role: 'editor',
        active: true,
        mfa_enabled: true,
        created_at: daysAgoIso(20),
        updated_at: daysAgoIso(2),
      },
      {
        id: 'usr_viewer_03',
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
      { id: 'mem_01', email: 'alice@acme.dev', name: 'Alice Chen', status: 'active', email_verified: true, created_at: daysAgoIso(14) },
      { id: 'mem_02', email: 'bob@acme.dev', name: 'Bob Miller', status: 'active', email_verified: true, created_at: daysAgoIso(9) },
      { id: 'mem_03', email: 'carol@acme.dev', name: 'Carol Diaz', status: 'active', email_verified: false, created_at: daysAgoIso(3) },
      { id: 'mem_04', email: 'dave@acme.dev', name: 'Dave Park', status: 'pending', email_verified: false, created_at: daysAgoIso(1) },
    ],
    media: [
      {
        id: 'med_01',
        original_filename: 'architecture-diagram.svg',
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
    ],
    audit: [
      { id: 'aud_01', action: 'auth.login', actor: 'admin@example.com', target: 'session', request_id: 'req_9f8a12bc34de', created_at: daysAgoIso(0), metadata: { ip: '127.0.0.1', method: 'POST', email: 'admin@example.com' } },
      { id: 'aud_02', action: 'news.publish', actor: 'admin@example.com', target: 'nw_051', request_id: 'req_7c41de90ab12', created_at: daysAgoIso(0), metadata: { ip: '127.0.0.1', method: 'PUT', email: 'admin@example.com' } },
      { id: 'aud_03', action: 'security.upload_quarantine_ok', actor: 'editor@example.com', target: 'med_02', request_id: 'req_5b2190ff11aa', created_at: daysAgoIso(1), metadata: { ip: '10.0.0.14', method: 'POST', email: 'editor@example.com' } },
      { id: 'aud_04', action: 'schema.update', actor: 'admin@example.com', target: 'changelog', request_id: 'req_3e1099cc8877', created_at: daysAgoIso(4), metadata: { ip: '127.0.0.1', method: 'PATCH', email: 'admin@example.com' } },
    ],
    notifications: [
      { id: 'ntf_01', title: 'v0.5.1 Live Demo Active', message: 'Changes are stored locally in your browser localStorage.', read: false, created_at: daysAgoIso(0) },
      { id: 'ntf_02', title: 'Security Check Passed', message: 'All 62 static routes and backend checks passed.', read: true, created_at: daysAgoIso(1) },
    ],
    site: {
      general: {
        site_name: 'HydroGo Demo Workspace',
        site_url: 'https://hydrogo.pages.dev',
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
      uptime_seconds: 86420,
      requests: 14820,
      views: 9412,
      errors: 2,
      avg_latency_ms: 4.2,
    },
    analytics: {
      unique_visitors_today: 418,
      daily: buildDailyAnalytics(),
    },
  };
}
