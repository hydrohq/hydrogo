import type { RequestHandler } from './$types';

function splitIntoChunks(text: string, maxLen = 180): string[] {
  const clean = (text || '').replace(/\s+/g, ' ').trim();
  if (!clean) return [];
  if (clean.length <= maxLen) return [clean];

  const sentences = clean.split(/(?<=[.?!;:\n])\s+/);
  const chunks: string[] = [];
  let current = '';

  for (const sentence of sentences) {
    if ((current + ' ' + sentence).trim().length <= maxLen) {
      current = (current + ' ' + sentence).trim();
    } else {
      if (current) chunks.push(current);
      if (sentence.length <= maxLen) {
        current = sentence;
      } else {
        // Fallback word split for very long sentences
        const words = sentence.split(' ');
        let wordChunk = '';
        for (const w of words) {
          if ((wordChunk + ' ' + w).trim().length <= maxLen) {
            wordChunk = (wordChunk + ' ' + w).trim();
          } else {
            if (wordChunk) chunks.push(wordChunk);
            wordChunk = w;
          }
        }
        current = wordChunk;
      }
    }
  }
  if (current) chunks.push(current);
  return chunks;
}

// In-memory rate limiting map: IP -> timestamp array
const ipRequests = new Map<string, number[]>();

function isRateLimited(clientIp: string, maxRequests = 10, windowMs = 60_000): boolean {
  const now = Date.now();
  const timestamps = (ipRequests.get(clientIp) || []).filter((ts) => now - ts < windowMs);
  if (timestamps.length >= maxRequests) {
    ipRequests.set(clientIp, timestamps);
    return true;
  }
  timestamps.push(now);
  ipRequests.set(clientIp, timestamps);
  return false;
}

export const GET: RequestHandler = async ({ url, request, getClientAddress }) => {
  // 1. IP rate limiting: max 10 requests per minute
  let clientIp = '127.0.0.1';
  try {
    clientIp = getClientAddress();
  } catch {
    clientIp = request.headers.get('x-forwarded-for')?.split(',')[0].trim() || 'unknown';
  }

  if (isRateLimited(clientIp, 10, 60_000)) {
    return new Response('Too Many Requests', { status: 429, headers: { 'Retry-After': '60' } });
  }

  // 2. Secret token check if configured
  const ttsSecret = process.env.TTS_API_SECRET;
  if (ttsSecret) {
    const auth = request.headers.get('Authorization') || url.searchParams.get('token');
    if (auth !== `Bearer ${ttsSecret}` && auth !== ttsSecret) {
      return new Response('Unauthorized', { status: 401 });
    }
  }

  const text = (url.searchParams.get('text') || '').trim();
  if (!text) {
    return new Response('Text is required', { status: 400 });
  }

  // 3. Length cap to prevent resource exhaustion (max 1000 characters)
  if (text.length > 1000) {
    return new Response('Text exceeds 1000 characters maximum limit', { status: 400 });
  }

  const chunks = splitIntoChunks(text, 180);
  if (chunks.length === 0) {
    return new Response('Empty text', { status: 400 });
  }

  try {
    // Fetch all audio chunks in parallel (capped at max 15 chunks to avoid abuse)
    const limitedChunks = chunks.slice(0, 15);
    const audioBuffers = await Promise.all(
      limitedChunks.map(async (c) => {
        const ttsUrl = `https://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&tl=tr&q=${encodeURIComponent(c)}`;
        const res = await fetch(ttsUrl, {
          headers: {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
            Referer: 'https://translate.google.com/',
          },
        });
        if (!res.ok) throw new Error(`Google TTS error: ${res.status}`);
        return res.arrayBuffer();
      })
    );

    // Concatenate all MP3 buffers
    const totalLength = audioBuffers.reduce((acc, buf) => acc + buf.byteLength, 0);
    const merged = new Uint8Array(totalLength);
    let offset = 0;
    for (const buf of audioBuffers) {
      merged.set(new Uint8Array(buf), offset);
      offset += buf.byteLength;
    }

    return new Response(merged, {
      headers: {
        'Content-Type': 'audio/mpeg',
        'Cache-Control': 'public, max-age=86400',
      },
    });
  } catch (err: any) {
    return new Response(err?.message || 'TTS Error', { status: 502 });
  }
};
