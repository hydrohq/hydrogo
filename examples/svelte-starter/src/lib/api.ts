export interface Page {
  id: string;
  title: string;
  slug: string;
  locale?: string;
  body?: string;
  blocks?: Array<{ type: string; payload: Record<string, any> }>;
  updated_at?: string;
}

export interface NewsArticle {
  id: string;
  title: string;
  slug: string;
  excerpt?: string;
  body?: string;
  cover_media_id?: string;
  author_name?: string;
  category?: string;
  tags?: string[];
  published_at?: string;
}

export interface MemberProfile {
  id: string;
  email: string;
  active: boolean;
}

const API_BASE = '/api/v1';
const DEFAULT_LOCALE = 'en';

export async function fetchPublishedPages(locale: string = DEFAULT_LOCALE): Promise<Page[]> {
  try {
    const res = await fetch(`${API_BASE}/pages?locale=${encodeURIComponent(locale)}`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.data || [];
  } catch (err) {
    console.warn('Failed to fetch pages:', err);
    return [];
  }
}

export async function fetchPageBySlug(slug: string, locale: string = DEFAULT_LOCALE): Promise<Page | null> {
  try {
    const res = await fetch(`${API_BASE}/pages/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`);
    if (!res.ok) return null;
    const data = await res.json();
    return data.data || null;
  } catch (err) {
    console.warn(`Failed to fetch page ${slug}:`, err);
    return null;
  }
}

export async function fetchPublishedNews(locale: string = DEFAULT_LOCALE): Promise<NewsArticle[]> {
  try {
    const res = await fetch(`${API_BASE}/news?locale=${encodeURIComponent(locale)}`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.data || [];
  } catch (err) {
    console.warn('Failed to fetch news:', err);
    return [];
  }
}

export async function fetchNewsBySlug(slug: string, locale: string = DEFAULT_LOCALE): Promise<NewsArticle | null> {
  try {
    const res = await fetch(`${API_BASE}/news/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`);
    if (!res.ok) return null;
    const data = await res.json();
    return data.data || null;
  } catch (err) {
    console.warn(`Failed to fetch news article ${slug}:`, err);
    return null;
  }
}
