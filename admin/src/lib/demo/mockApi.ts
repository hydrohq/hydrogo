import { createInitialDemoState, buildDashboardPayload, type DemoState } from './seedData';

const STORAGE_KEY = 'hydrogo_admin_demo_v1';

function loadState(): DemoState {
  if (typeof window === 'undefined') return createInitialDemoState();
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as DemoState;
      if (parsed && parsed.schema && parsed.user) {
        return parsed;
      }
    }
  } catch {
    // Fallback to fresh seed if localStorage is corrupted
  }
  const fresh = createInitialDemoState();
  saveState(fresh);
  return fresh;
}

function saveState(state: DemoState): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Ignore quota errors in private browsing
  }
}

export function resetDemoState(): void {
  const fresh = createInitialDemoState();
  saveState(fresh);
  if (typeof window !== 'undefined') {
    window.location.reload();
  }
}

function parseBody(body: BodyInit | null | undefined): any {
  if (!body) return {};
  if (typeof body === 'string') {
    try {
      return JSON.parse(body);
    } catch {
      const params = new URLSearchParams(body);
      return Object.fromEntries(params.entries());
    }
  }
  return {};
}

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}`;
}

function recordAudit(state: DemoState, action: string, target: string, method: string): void {
  state.audit.unshift({
    id: newId('aud'),
    action,
    actor: state.user.email,
    target,
    request_id: newId('req'),
    created_at: new Date().toISOString(),
    metadata: { ip: '127.0.0.1', method, email: state.user.email },
  });
  if (state.audit.length > 50) state.audit.length = 50;
}

function crudList(
  items: Array<Record<string, any>>,
  method: string,
  subId: string | undefined,
  payload: any,
  state: DemoState,
  prefix: string,
  entityName: string
): any {
  if (method === 'GET') {
    if (subId) {
      const found = items.find((item) => item.id === subId || item.slug === subId);
      if (!found) {
        const err: any = new Error(`${entityName} not found`);
        err.status = 404;
        throw err;
      }
      return { data: found };
    }
    return { data: items, items, total: items.length };
  }

  if (method === 'POST' && !subId) {
    const now = new Date().toISOString();
    const created = {
      id: newId(prefix),
      status: 'draft',
      created_at: now,
      updated_at: now,
      ...payload,
    };
    items.unshift(created);
    recordAudit(state, `${entityName}.create`, created.id, 'POST');
    saveState(state);
    return { data: created };
  }

  if ((method === 'PUT' || method === 'PATCH') && subId) {
    const idx = items.findIndex((item) => item.id === subId);
    if (idx === -1) {
      const err: any = new Error(`${entityName} not found`);
      err.status = 404;
      throw err;
    }
    items[idx] = { ...items[idx], ...payload, updated_at: new Date().toISOString() };
    recordAudit(state, `${entityName}.update`, subId, method);
    saveState(state);
    return { data: items[idx] };
  }

  if (method === 'DELETE' && subId) {
    const idx = items.findIndex((item) => item.id === subId);
    if (idx !== -1) {
      items.splice(idx, 1);
      recordAudit(state, `${entityName}.delete`, subId, 'DELETE');
      saveState(state);
    }
    return { ok: true };
  }

  return { data: items, total: items.length };
}

export async function handleDemoRequest<T>(
  endpoint: string,
  options: { method?: string; body?: BodyInit | null; noAuthRedirect?: boolean } = {}
): Promise<T> {
  const method = (options.method || 'GET').toUpperCase();
  const path = endpoint.split('?')[0].replace(/\/+$/, '') || '/';
  const state = loadState();
  const payload = parseBody(options.body);

  // Auth endpoints
  if (path === '/auth/login' && method === 'POST') {
    state.authenticated = true;
    recordAudit(state, 'auth.login', 'session', 'POST');
    saveState(state);
    return { ok: true, data: { user: state.user } } as T;
  }
  if (path === '/auth/logout' && method === 'POST') {
    state.authenticated = false;
    saveState(state);
    return { ok: true } as T;
  }

  if (!state.authenticated && path === '/api/v1/admin/me') {
    const err: any = new Error('Unauthorized');
    err.status = 401;
    err.code = 'UNAUTHORIZED';
    throw err;
  }

  if (path === '/api/v1/admin/me') {
    return {
      data: {
        user: state.user,
        permissions: ['*'],
        locale: null,
        csrf_token: 'hyd_demo_csrf_token',
      },
    } as T;
  }

  if (path === '/api/v1/admin/schema') {
    return { data: state.schema } as T;
  }

  if (path === '/api/v1/admin/dashboard') {
    return { data: buildDashboardPayload(state) } as T;
  }

  if (path === '/api/v1/admin/dashboard/recent') {
    const recent = [
      ...state.pages.map((p) => ({ ...p, kind: 'page' })),
      ...state.news.map((n) => ({ ...n, kind: 'news' })),
    ].slice(0, 10);
    return { data: recent, items: recent, total: recent.length } as T;
  }

  if (path === '/api/v1/admin/audit') {
    return { data: state.audit, items: state.audit, total: state.audit.length } as T;
  }

  if (path === '/api/v1/admin/notifications') {
    return { data: state.notifications, items: state.notifications, unread: state.notifications.filter((n) => !n.read).length } as T;
  }
  if (path === '/api/v1/admin/notifications/read-all' && method === 'POST') {
    state.notifications.forEach((n) => { n.read = true; });
    saveState(state);
    return { ok: true } as T;
  }

  if (path === '/api/v1/admin/roles') {
    return {
      data: [
        { id: 'admin', name: 'Administrator', permissions: ['*'] },
        { id: 'editor', name: 'Editor', permissions: ['content:read', 'content:write', 'media:write'] },
        { id: 'viewer', name: 'Viewer', permissions: ['content:read', 'audit:read'] },
      ],
    } as T;
  }

  if (path.startsWith('/api/v1/admin/pages')) {
    const subId = path.split('/')[5];
    return crudList(state.pages, method, subId, payload, state, 'pg', 'pages') as T;
  }

  if (path.startsWith('/api/v1/admin/news')) {
    const subId = path.split('/')[5];
    if (subId === 'categories' || subId === 'tags') {
      return { data: [] } as T;
    }
    return crudList(state.news, method, subId, payload, state, 'nw', 'news') as T;
  }

  if (path.startsWith('/api/v1/admin/content/')) {
    const parts = path.split('/');
    const typeId = parts[5];
    const subId = parts[6];
    if (!state.content[typeId]) state.content[typeId] = [];
    return crudList(state.content[typeId], method, subId, payload, state, typeId.slice(0, 3), typeId) as T;
  }

  if (path.startsWith('/api/v1/admin/users')) {
    const subId = path.split('/')[5];
    return crudList(state.users, method, subId, payload, state, 'usr', 'users') as T;
  }

  if (path.startsWith('/api/v1/admin/members')) {
    const subId = path.split('/')[5];
    return crudList(state.members, method, subId, payload, state, 'mem', 'members') as T;
  }

  if (path.startsWith('/api/v1/admin/media')) {
    const subId = path.split('/')[5];
    return crudList(state.media, method, subId, payload, state, 'med', 'media') as T;
  }

  if (path.startsWith('/api/v1/admin/site')) {
    const section = path.split('/')[5] || 'general';
    if (method === 'PUT' || method === 'POST' || method === 'PATCH') {
      state.site[section] = { ...(state.site[section] || {}), ...payload };
      recordAudit(state, `site.${section}.update`, section, method);
      saveState(state);
    }
    return { data: state.site[section] || {} } as T;
  }

  if (path.startsWith('/api/v1/admin/system')) {
    return {
      data: {
        version: 'v0.5.1',
        go_version: 'go1.27',
        driver: 'browser-localstorage-demo',
        status: 'healthy',
        uptime_seconds: 86420,
      },
    } as T;
  }

  if (method === 'GET') {
    return { data: [], items: [], total: 0 } as T;
  }
  return { ok: true, data: payload } as T;
}
