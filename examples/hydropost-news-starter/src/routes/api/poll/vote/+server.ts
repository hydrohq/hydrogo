import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const API_BASE = (typeof process !== 'undefined' && process.env?.HYDROGO_API_URL) || 'http://127.0.0.1:8090';
const API_KEY = (typeof process !== 'undefined' && (process.env?.HYDROGO_API_KEY || process.env?.HYDROGO_ADMIN_KEY)) || '';

// In-memory rate limiting map (IP -> timestamp)
const recentVotes = new Map<string, number>();

export const POST: RequestHandler = async ({ request, getClientAddress, cookies }) => {
  let body: { pollId?: string; option?: string };
  try {
    body = await request.json();
  } catch {
    return json({ code: 'bad_request', message: 'Geçersiz JSON verisi.' }, { status: 400 });
  }

  const { pollId, option } = body;
  if (!pollId || !option) {
    return json({ code: 'bad_request', message: 'Anket ID ve seçenek zorunludur.' }, { status: 400 });
  }

  // Check double-voting via cookie
  const cookieKey = `voted_${pollId}`;
  if (cookies.get(cookieKey)) {
    return json({ code: 'already_voted', message: 'Bu ankete daha önce oy kullandınız.' }, { status: 409 });
  }

  // Rate-limit by client IP (max 1 vote per 2 seconds per IP)
  let clientIp = 'unknown';
  try {
    clientIp = getClientAddress();
  } catch {
    // getClientAddress might throw in some adapter environments
  }
  const now = Date.now();
  const lastVote = recentVotes.get(`${clientIp}_${pollId}`);
  if (lastVote && now - lastVote < 2000) {
    return json({ code: 'too_many_requests', message: 'Lütfen birkaç saniye bekleyin.' }, { status: 429 });
  }
  recentVotes.set(`${clientIp}_${pollId}`, now);

  // Set 30-day cookie to record vote on browser
  cookies.set(cookieKey, option, {
    path: '/',
    maxAge: 30 * 24 * 60 * 60,
    httpOnly: false,
    sameSite: 'lax',
  });

  // Handle demo / fallback poll
  if (pollId.startsWith('demo-')) {
    const demoTallies: Record<string, number> = {
      Galatasaray: 620,
      Fenerbahçe: 480,
      Beşiktaş: 230,
      Trabzonspor: 110,
      Diğer: 42,
    };
    demoTallies[option] = (demoTallies[option] || 0) + 1;
    const total = Object.values(demoTallies).reduce((acc, v) => acc + v, 0);

    const percentages: Record<string, number> = {};
    for (const [k, v] of Object.entries(demoTallies)) {
      percentages[k] = Math.round((v / total) * 100);
    }

    return json({
      success: true,
      total_votes: total,
      votes_data: demoTallies,
      percentages,
    });
  }

  // Fetch real poll from HydroGo
  try {
    const getRes = await fetch(`${API_BASE}/api/v1/content/polls/${pollId}`, {
      headers: { Accept: 'application/json' },
    });

    if (!getRes.ok) {
      return json({ code: 'poll_not_found', message: 'Anket bulunamadı.' }, { status: 404 });
    }

    const jsonRes = await getRes.json();
    const pollRecord = jsonRes.data;

    let tallies: Record<string, number> = {};
    if (pollRecord.votes_data) {
      try {
        tallies = JSON.parse(pollRecord.votes_data);
      } catch {
        tallies = {};
      }
    }

    // Increment selected option
    tallies[option] = (tallies[option] || 0) + 1;
    const newTotal = (pollRecord.total_votes || 0) + 1;

    // Persist to HydroGo via Admin API
    const updateHeaders: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    };
    if (API_KEY) {
      updateHeaders['X-API-Key'] = API_KEY;
    }

    const putRes = await fetch(`${API_BASE}/api/v1/admin/content/polls/${pollId}`, {
      method: 'PUT',
      headers: updateHeaders,
      body: JSON.stringify({
        ...pollRecord,
        total_votes: newTotal,
        votes_data: JSON.stringify(tallies),
      }),
    });

    if (!putRes.ok) {
      console.warn(`[POLL VOTE] HydroGo admin update returned status ${putRes.status}`);
    }

    // Calculate real percentages
    const percentages: Record<string, number> = {};
    for (const [k, v] of Object.entries(tallies)) {
      percentages[k] = newTotal > 0 ? Math.round((v / newTotal) * 100) : 0;
    }

    return json({
      success: true,
      total_votes: newTotal,
      votes_data: tallies,
      percentages,
    });
  } catch (err) {
    console.error('[POLL VOTE] Error updating poll vote:', err);
    return json({ code: 'internal_error', message: 'Oy kaydedilirken bir hata oluştu.' }, { status: 500 });
  }
};

// Lets a reader take back their own vote: reads which option they picked
// from the vote cookie (set in POST above), decrements that tally, clears
// the cookie so they could vote again.
export const DELETE: RequestHandler = async ({ request, cookies }) => {
  let body: { pollId?: string };
  try {
    body = await request.json();
  } catch {
    return json({ code: 'bad_request', message: 'Geçersiz JSON verisi.' }, { status: 400 });
  }

  const { pollId } = body;
  if (!pollId) {
    return json({ code: 'bad_request', message: 'Anket ID zorunludur.' }, { status: 400 });
  }

  const cookieKey = `voted_${pollId}`;
  const votedOption = cookies.get(cookieKey);
  if (!votedOption) {
    return json({ code: 'not_voted', message: 'Bu ankete oy vermediniz.' }, { status: 409 });
  }
  cookies.delete(cookieKey, { path: '/' });

  if (pollId.startsWith('demo-')) {
    return json({ success: true });
  }

  try {
    const getRes = await fetch(`${API_BASE}/api/v1/content/polls/${pollId}`, { headers: { Accept: 'application/json' } });
    if (!getRes.ok) {
      return json({ success: true }); // vote cookie cleared either way
    }
    const jsonRes = await getRes.json();
    const pollRecord = jsonRes.data;

    let tallies: Record<string, number> = {};
    if (pollRecord.votes_data) {
      try {
        tallies = JSON.parse(pollRecord.votes_data);
      } catch {
        tallies = {};
      }
    }
    tallies[votedOption] = Math.max(0, (tallies[votedOption] || 1) - 1);
    const newTotal = Math.max(0, (pollRecord.total_votes || 1) - 1);

    const updateHeaders: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'application/json' };
    if (API_KEY) updateHeaders['X-API-Key'] = API_KEY;

    await fetch(`${API_BASE}/api/v1/admin/content/polls/${pollId}`, {
      method: 'PUT',
      headers: updateHeaders,
      body: JSON.stringify({ ...pollRecord, total_votes: newTotal, votes_data: JSON.stringify(tallies) }),
    });

    return json({ success: true, total_votes: newTotal, votes_data: tallies });
  } catch (err) {
    console.error('[POLL VOTE] Error removing poll vote:', err);
    return json({ success: true });
  }
};
