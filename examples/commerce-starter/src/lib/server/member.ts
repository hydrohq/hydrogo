import type { Cookies } from '@sveltejs/kit';
import { proxiedFetch } from '$lib/server/apiProxy';

// modules/members' public /member/* endpoints don't use the {data,...}
// envelope commerce's API does (register/login return {authenticated,
// redirect, member} directly; errors are plain text, not JSON) -- so this
// module interprets responses itself rather than reusing commerce.ts's
// request helper.
export class MemberAPIError extends Error {
  constructor(readonly status: number, message: string) {
    super(message);
    this.name = 'MemberAPIError';
  }
}

export type Member = { id: string; email: string; active: boolean };

async function memberAuthRequest(cookies: Cookies, path: string, body: Record<string, string>): Promise<Member> {
  const response = await proxiedFetch(cookies, path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  const text = await response.text();
  if (!response.ok) throw new MemberAPIError(response.status, text || `HydroGo member API returned ${response.status}`);
  const parsed = JSON.parse(text) as { member: Member };
  return parsed.member;
}

export async function registerMember(cookies: Cookies, email: string, password: string): Promise<Member> {
  return memberAuthRequest(cookies, '/member/register', { email, password });
}

export async function loginMember(cookies: Cookies, email: string, password: string): Promise<Member> {
  return memberAuthRequest(cookies, '/member/login', { email, password });
}

export async function logoutMember(cookies: Cookies): Promise<void> {
  await proxiedFetch(cookies, '/member/logout', { method: 'POST' });
}

/** The signed-in member, or null when not logged in -- never throws, since
 * every page's layout calls this to decide what to show in the header. */
export async function currentMember(cookies: Cookies): Promise<Member | null> {
  try {
    const response = await proxiedFetch(cookies, '/member/me');
    if (!response.ok) return null;
    return (await response.json()) as Member;
  } catch {
    return null;
  }
}

export type MemberOrderSummary = {
  id: string;
  currency: string;
  total_cents: number;
  payment_status: string;
  shipping_status: string;
};

export async function listMemberOrders(cookies: Cookies): Promise<MemberOrderSummary[]> {
  const response = await proxiedFetch(cookies, '/member/orders');
  if (!response.ok) return [];
  const text = await response.text();
  return text ? (JSON.parse(text) as MemberOrderSummary[]) : [];
}
