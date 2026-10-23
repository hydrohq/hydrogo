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

function requireString(entry: Record<string, unknown>, field: string, index: number): string {
  const value = entry[field];
  if (typeof value !== 'string' || !value) throw new Error(`CRM_TENANTS[${index}].${field} must be a non-empty string`);
  return value;
}

function parseTenants(raw: string): Tenant[] {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (cause) {
    throw new Error(`CRM_TENANTS is not valid JSON: ${(cause as Error).message}`);
  }
  if (!Array.isArray(parsed)) throw new Error('CRM_TENANTS must be a JSON array of tenant objects');
  return parsed.map((value, index) => {
    const entry = value as Record<string, unknown>;
    const slug = requireString(entry, 'slug', index);
    const host = requireString(entry, 'host', index);
    const hydroOrigin = requireString(entry, 'hydroOrigin', index).replace(/\/$/, '');
    const webhookSecret = entry.webhookSecret;
    if (webhookSecret !== undefined && typeof webhookSecret !== 'string') {
      throw new Error(`CRM_TENANTS[${index}].webhookSecret must be a string`);
    }
    return { id: slug, slug, host, hydroOrigin, webhookSecret };
  });
}

let cached: Tenant[] | null = null;

/** Parses CRM_TENANTS once per process: a JSON array of {slug, host, hydroOrigin, webhookSecret}. */
function configuredTenants(): Tenant[] {
  if (cached) return cached;
  cached = env.CRM_TENANTS ? parseTenants(env.CRM_TENANTS) : [];
  return cached;
}

// Validate at module load (server startup), not on the first request: a bad CRM_TENANTS
// should crash loudly in the startup log, not turn into an opaque 500 on every request
// once the production error page has hidden the real message.
configuredTenants();

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
