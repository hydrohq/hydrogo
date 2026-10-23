import { env } from '$env/dynamic/private';

export interface Tenant {
  id: string;
  slug: string;
  host: string;
  hydroOrigin: string;
  webhookSecret?: string;
}

function normalizeHost(host: string): string {
  return host.split(':')[0].trim().toLowerCase();
}

function defaultOrigin(): string {
  return (env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
}

let cached: Tenant[] | null = null;

/** Parses CRM_TENANTS once per process: a JSON array of {slug, host, hydroOrigin, webhookSecret}. */
function configuredTenants(): Tenant[] {
  if (cached) return cached;
  const raw = env.CRM_TENANTS;
  cached = raw
    ? (JSON.parse(raw) as Omit<Tenant, 'id'>[]).map((tenant) => ({ ...tenant, id: tenant.slug }))
    : [];
  return cached;
}

/**
 * Resolves which tenant a request belongs to, HydroGo-style: by Host header.
 * With no CRM_TENANTS configured, every request is the single default tenant
 * (HYDRO_API_ORIGIN) — today's single-tenant behaviour, unchanged.
 */
export function resolveTenant(hostHeader: string | null): Tenant | null {
  const tenants = configuredTenants();
  if (tenants.length === 0) return { id: 'default', slug: 'default', host: '*', hydroOrigin: defaultOrigin() };
  if (!hostHeader) return null;
  const host = normalizeHost(hostHeader);
  return tenants.find((tenant) => normalizeHost(tenant.host) === host) ?? null;
}

/** For tests. */
export function resetTenantCache(): void {
  cached = null;
}
