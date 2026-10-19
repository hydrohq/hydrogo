/**
 * Client-side view tracker with persistent device deduplication.
 * Prevents multiple view counts from the same browser/device.
 */
export async function trackView(slug: string, type: 'news' | 'page' = 'news'): Promise<number | null> {
  if (typeof window === 'undefined' || !slug) return null;

  const storageKey = `hp_read_${slug}`;
  const alreadyRead = localStorage.getItem(storageKey);

  // If already viewed on this device, skip increment request
  if (alreadyRead) {
    return null;
  }

  try {
    const res = await fetch('/api/v1/news/views', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ slug, type }),
    });

    if (res.ok) {
      const json = await res.json();
      const data = json.data || json;
      localStorage.setItem(storageKey, Date.now().toString());
      return typeof data.views === 'number' ? data.views : null;
    }
  } catch (err) {
    console.warn('View tracking network error:', err);
  }

  return null;
}

export function formatViews(count?: number | null): string {
  if (!count && count !== 0) return '0';
  return count.toLocaleString('tr-TR');
}

function getDeterministicBaseline(slug: string): number {
  let hash = 0;
  for (let i = 0; i < slug.length; i++) {
    hash = (hash << 5) - hash + slug.charCodeAt(i);
    hash |= 0;
  }
  const pos = Math.abs(hash);
  return (pos % 3800) + 420;
}

export function getViewCount(slug: string): number {
  if (!slug) return 0;
  if (typeof window !== 'undefined') {
    const stored = localStorage.getItem(`views_${slug}`);
    if (stored) return parseInt(stored, 10);
  }
  return getDeterministicBaseline(slug);
}

export function incrementViewCount(slug: string): number {
  if (!slug) return 0;
  const current = getViewCount(slug);
  const next = current + 1;
  if (typeof window !== 'undefined') {
    localStorage.setItem(`views_${slug}`, next.toString());
  }
  return next;
}
