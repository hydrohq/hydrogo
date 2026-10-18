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
    const res = await fetch('/api/views', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ slug, type }),
    });

    if (res.ok) {
      const data = await res.json();
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
