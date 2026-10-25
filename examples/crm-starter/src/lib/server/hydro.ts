import type { Session } from './sessions';

/** An error envelope from HydroGo: {code, message, request_id, fields}. */
export class HydroError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public requestId: string,
    public fields?: Record<string, string>
  ) {
    super(message);
  }
}

export interface Record_ {
  id: string;
  version: number;
  created_at: string;
  updated_at: string;
  expanded?: Record<string, Record_ | Record_[]>;
  [field: string]: unknown;
}

export interface List {
  data: Record_[];
  metadata: { limit: number; has_more: boolean; next_cursor?: string };
}

function parseCookies(headers: Headers): Record<string, string> {
  const result: Record<string, string> = {};
  for (const line of headers.getSetCookie()) {
    const [pair] = line.split(';');
    const index = pair.indexOf('=');
    if (index > 0) result[pair.slice(0, index).trim()] = pair.slice(index + 1).trim();
  }
  return result;
}

async function fail(res: Response): Promise<never> {
  let body: { code?: string; message?: string; request_id?: string; fields?: Record<string, string> } = {};
  try {
    body = await res.json();
  } catch {
    /* plain-text error */
  }
  throw new HydroError(res.status, body.code ?? 'error', body.message ?? res.statusText, body.request_id ?? '', body.fields);
}

export type LoginResult =
  | { ok: true; cookie: string; csrf: string }
  | { ok: false; mfaRequired?: boolean; message: string };

/** Sign in against HydroGo (CSRF + form login). Returns the server-side cookie pair. */
export async function login(origin: string, email: string, password: string, mfaCode: string): Promise<LoginResult> {
  const base = origin.replace(/\/$/, '');
  const csrfRes = await fetch(`${base}/csrf-token`);
  const csrfCookie = parseCookies(csrfRes.headers)['hydro_csrf'];
  const { token } = (await csrfRes.json()) as { token: string };
  const form = new URLSearchParams({ email, password });
  if (mfaCode) form.set('mfa_code', mfaCode);
  const res = await fetch(`${base}/auth/login`, {
    method: 'POST',
    headers: {
      'content-type': 'application/x-www-form-urlencoded',
      'x-csrf-token': token,
      cookie: `hydro_csrf=${csrfCookie}`
    },
    body: form
  });
  if (res.status === 401) {
    const text = await res.text();
    if (text.includes('mfa_code_required')) return { ok: false, mfaRequired: true, message: 'mfa' };
    return { ok: false, message: 'invalid' };
  }
  if (!res.ok) return { ok: false, message: res.status === 429 ? 'rate' : 'unavailable' };
  const session = parseCookies(res.headers)['hydro_session'];
  if (!session) return { ok: false, message: 'unavailable' };
  return { ok: true, cookie: `hydro_session=${session}; hydro_csrf=${csrfCookie}`, csrf: token };
}

export async function logout(session: Session) {
  await fetch(`${session.hydroOrigin}/auth/logout`, {
    method: 'POST',
    headers: { cookie: session.cookie, 'x-csrf-token': session.csrf }
  }).catch(() => undefined);
}

async function call(session: Session, method: string, path: string, body?: unknown): Promise<Response> {
  const headers: Record<string, string> = { cookie: session.cookie, accept: 'application/json' };
  if (method !== 'GET') headers['x-csrf-token'] = session.csrf;
  if (body !== undefined) headers['content-type'] = 'application/json';
  return fetch(`${session.hydroOrigin}${path}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
}

export interface Query {
  limit?: number;
  search?: string;
  sort?: string;
  expand?: string[];
  /** filter[field]=value or filter[field][op]=value */
  filter?: Record<string, string>;
  after?: string;
}

export async function list(session: Session, type: string, q: Query = {}): Promise<List> {
  const params = new URLSearchParams();
  params.set('limit', String(q.limit ?? 50));
  if (q.search) params.set('search', q.search);
  if (q.sort) params.set('sort', q.sort);
  if (q.expand?.length) params.set('expand', q.expand.join(','));
  if (q.after) params.set('after', q.after);
  for (const [key, value] of Object.entries(q.filter ?? {})) params.set(`filter[${key}]`, value);
  const res = await call(session, 'GET', `/api/v1/admin/content/${type}?${params}`);
  if (!res.ok) return fail(res);
  return res.json();
}

export interface Aggregate {
  total: { count: number; sums: Record<string, string> };
  groups: { key: string | null; count: number; sums: Record<string, string> }[];
}

/**
 * Exact counts and sums over every matching record (not just one page), via
 * GET .../_aggregate. Sums arrive as exact decimal strings.
 */
export async function aggregate(session: Session, type: string, q: { groupBy?: string; sum?: string[]; filter?: Record<string, string> } = {}): Promise<Aggregate> {
  const params = new URLSearchParams();
  if (q.groupBy) params.set('group_by', q.groupBy);
  if (q.sum?.length) params.set('sum', q.sum.join(','));
  for (const [key, value] of Object.entries(q.filter ?? {})) params.set(`filter[${key}]`, value);
  const res = await call(session, 'GET', `/api/v1/admin/content/${type}/_aggregate?${params}`);
  if (!res.ok) return fail(res);
  const body = (await res.json()).data as Partial<Aggregate>;
  return { total: body.total ?? { count: 0, sums: {} }, groups: body.groups ?? [] };
}

export async function get(session: Session, type: string, id: string, expand: string[] = []): Promise<Record_> {
  const query = expand.length ? `?expand=${expand.join(',')}` : '';
  const res = await call(session, 'GET', `/api/v1/admin/content/${type}/${encodeURIComponent(id)}${query}`);
  if (!res.ok) return fail(res);
  return (await res.json()).data;
}

export async function create(session: Session, type: string, fields: Record<string, unknown>): Promise<Record_> {
  const res = await call(session, 'POST', `/api/v1/admin/content/${type}`, fields);
  if (!res.ok) return fail(res);
  return (await res.json()).data;
}

/** Update with optimistic locking: pass the record's current version. */
export async function update(session: Session, type: string, id: string, version: number, fields: Record<string, unknown>): Promise<Record_> {
  const res = await call(session, 'PUT', `/api/v1/admin/content/${type}/${encodeURIComponent(id)}`, { version, ...fields });
  if (!res.ok) return fail(res);
  return (await res.json()).data;
}

export async function me(session: Session): Promise<{ email?: string; permissions: string[] } | null> {
  const res = await call(session, 'GET', '/api/v1/admin/me');
  if (res.status === 401) return null;
  if (!res.ok) return fail(res);
  const data = (await res.json()).data;
  return { email: data.user?.email ?? data.email, permissions: data.permissions ?? [] };
}

const SYSTEM_FIELDS = new Set(['id', 'version', 'created_at', 'updated_at', 'expanded']);

/** Fields of a record without the system keys, ready to send back in a PUT. */
export function fieldsOf(record: Record_): Record<string, unknown> {
  return Object.fromEntries(Object.entries(record).filter(([key]) => !SYSTEM_FIELDS.has(key)));
}

/** HydroGo updates replace the whole record, so merge the change into the current fields. */
export async function patch(session: Session, type: string, id: string, changes: Record<string, unknown>): Promise<Record_> {
  const current = await get(session, type, id);
  const merged: Record<string, unknown> = { ...fieldsOf(current), ...changes };
  for (const key of Object.keys(merged)) if (merged[key] === '' || merged[key] === null) delete merged[key];
  return update(session, type, id, current.version, merged);
}
