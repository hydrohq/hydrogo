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

export async function fetchPublishedPages(): Promise<Page[]> {
  try {
    const res = await fetch(`${API_BASE}/pages`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.data || [];
  } catch (err) {
    console.warn('Failed to fetch pages:', err);
    return [];
  }
}

export async function fetchPageBySlug(slug: string): Promise<Page | null> {
  try {
    const res = await fetch(`${API_BASE}/pages/${slug}`);
    if (!res.ok) return null;
    const data = await res.json();
    return data.data || null;
  } catch (err) {
    console.warn(`Failed to fetch page ${slug}:`, err);
    return null;
  }
}

export async function fetchPublishedNews(): Promise<NewsArticle[]> {
  try {
    const res = await fetch(`${API_BASE}/news`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.data || [];
  } catch (err) {
    console.warn('Failed to fetch news:', err);
    return [];
  }
}

export async function fetchNewsBySlug(slug: string): Promise<NewsArticle | null> {
  try {
    const res = await fetch(`${API_BASE}/news/${slug}`);
    if (!res.ok) return null;
    const data = await res.json();
    return data.data || null;
  } catch (err) {
    console.warn(`Failed to fetch news article ${slug}:`, err);
    return null;
  }
}
