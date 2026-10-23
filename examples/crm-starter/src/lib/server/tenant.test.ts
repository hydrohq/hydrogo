import { afterEach, describe, expect, it, vi } from 'vitest';

// $env/dynamic/private is a snapshot taken once at Vite startup, not a live view of
// process.env, so vi.stubEnv doesn't reach it — mock the module instead.
const mockEnv: Record<string, string | undefined> = {};
vi.mock('$env/dynamic/private', () => ({ env: mockEnv }));

const { resetTenantCache, resolveTenant } = await import('./tenant');

afterEach(() => {
  for (const key of Object.keys(mockEnv)) delete mockEnv[key];
  resetTenantCache();
});

describe('resolveTenant', () => {
  it('falls back to a single default tenant when CRM_TENANTS is unset', () => {
    mockEnv.HYDRO_API_ORIGIN = 'http://127.0.0.1:9000';
    expect(resolveTenant('anything.example.com')).toMatchObject({ id: 'default', hydroOrigin: 'http://127.0.0.1:9000' });
    expect(resolveTenant(null)).toMatchObject({ id: 'default' });
  });

  it('resolves by Host header, ignoring port and case, when tenants are configured', () => {
    mockEnv.CRM_TENANTS = JSON.stringify([{ slug: 'acme', host: 'acme.crm.example.com', hydroOrigin: 'https://acme.hydro.example.com' }]);
    expect(resolveTenant('ACME.crm.example.com:3000')).toMatchObject({ id: 'acme', hydroOrigin: 'https://acme.hydro.example.com' });
  });

  it('fails closed for an unrecognized or missing host', () => {
    mockEnv.CRM_TENANTS = JSON.stringify([{ slug: 'acme', host: 'acme.crm.example.com', hydroOrigin: 'https://acme.hydro.example.com' }]);
    expect(resolveTenant('other.example.com')).toBeNull();
    expect(resolveTenant(null)).toBeNull();
  });
});
