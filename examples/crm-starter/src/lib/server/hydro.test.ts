import { afterEach, describe, expect, it, vi } from 'vitest';
import { HydroError, fieldsOf, list, login } from './hydro';

afterEach(() => vi.restoreAllMocks());

function cookies(...lines: string[]) {
  const headers = new Headers();
  for (const line of lines) headers.append('set-cookie', line);
  return headers;
}

describe('login', () => {
  it('keeps HydroGo cookies and the CSRF token for the server-side session', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ token: 'tok' }), { headers: cookies('hydro_csrf=tok; Path=/') }))
      .mockResolvedValueOnce(new Response('{}', { headers: cookies('hydro_session=sess; Path=/; HttpOnly') }));
    const result = await login('a@b.co', 'pw', '');
    expect(result).toEqual({ ok: true, cookie: 'hydro_session=sess; hydro_csrf=tok', csrf: 'tok' });
    const [, init] = fetchMock.mock.calls[1];
    expect((init?.headers as Record<string, string>)['x-csrf-token']).toBe('tok');
  });

  it('asks for the MFA code when HydroGo does', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ token: 'tok' }), { headers: cookies('hydro_csrf=tok') }))
      .mockResolvedValueOnce(new Response('{"error":"mfa_code_required"}', { status: 401 }));
    expect(await login('a@b.co', 'pw', '')).toMatchObject({ ok: false, mfaRequired: true });
  });

  it('reports wrong credentials without leaking details', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ token: 'tok' }), { headers: cookies('hydro_csrf=tok') }))
      .mockResolvedValueOnce(new Response('invalid credentials', { status: 401 }));
    expect(await login('a@b.co', 'bad', '')).toEqual({ ok: false, message: 'invalid' });
  });
});

describe('api calls', () => {
  const session = { cookie: 'c', csrf: 't', email: 'a@b.co', expires: Date.now() + 1000 };

  it('turns the error envelope into a HydroError with the request id', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ code: 'unprocessable_entity', message: 'validation failed', request_id: 'r1', fields: { name: 'required' } }), { status: 422 }));
    const error = await list(session, 'deals').catch((e) => e);
    expect(error).toBeInstanceOf(HydroError);
    expect(error).toMatchObject({ status: 422, requestId: 'r1', fields: { name: 'required' } });
  });

  it('builds filter and expand parameters', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{"data":[],"metadata":{"limit":5,"has_more":false}}'));
    await list(session, 'deals', { limit: 5, filter: { stage: 'won' }, expand: ['company'], sort: '-created_at' });
    const url = String(fetchMock.mock.calls[0][0]);
    expect(decodeURIComponent(url)).toContain('filter[stage]=won');
    expect(url).toContain('expand=company');
    expect(url).toContain('limit=5');
  });

  it('strips system keys before a record is sent back', () => {
    expect(fieldsOf({ id: 'x', version: 2, created_at: '', updated_at: '', name: 'A', status: 'active' })).toEqual({ name: 'A', status: 'active' });
  });
});
