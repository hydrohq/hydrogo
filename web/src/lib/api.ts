export interface NewsArticle {
  id: string;
  locale: string;
  slug: string;
  title: string;
  body: string;
  cover_media_id?: string;
  excerpt?: string;
  key_takeaways?: string;
  category?: string;
  author_name?: string;
  featured?: boolean;
  tags?: string;
  seo_title?: string;
  seo_description?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Columnist {
  id: string;
  name: string;
  title?: string;
  avatar?: string;
  email?: string;
  bio?: string;
  latest_article_title?: string;
  latest_article_body?: string;
  sort_order?: number;
}

export interface Ad {
  id: string;
  name: string;
  zone: 'header_top' | 'manset_alti' | 'sur_manset_yani' | 'haber_ici' | 'sidebar' | 'footer';
  banner_image?: string;
  mobile_image?: string;
  target_url?: string;
  custom_code?: string;
  start_date?: string;
  end_date?: string;
}

export interface Gallery {
  id: string;
  title: string;
  slug: string;
  cover_image: string;
  description?: string;
  category?: string;
}

export interface VideoNews {
  id: string;
  title: string;
  slug: string;
  cover_image: string;
  video_url: string;
  duration?: string;
  category?: string;
}

export interface Masthead {
  id?: string;
  site_name: string;
  publisher: string;
  managing_editor: string;
  news_director?: string;
  legal_counsel?: string;
  hosting_provider: string;
  address: string;
  phone: string;
  email: string;
  kep_address?: string;
  publication_type?: string;
  editorial_principles?: string;
}

export interface OfficialAnnouncement {
  id: string;
  title: string;
  bik_number?: string;
  institution: string;
  announcement_type?: string;
  city?: string;
  deadline?: string;
  body: string;
  official_url?: string;
  created_at?: string;
}

export interface Poll {
  id: string;
  question: string;
  options: string;
  category?: string;
  total_votes?: number;
  is_active?: boolean;
  end_date?: string;
}

export interface ApiResponse<T> {
  data: T;
  metadata?: {
    limit?: number;
    offset?: number;
    total?: number;
    has_more?: boolean;
    next_cursor?: string;
  };
}

const API_BASE = (typeof process !== 'undefined' && process.env?.HYDROGO_API_URL) || 'http://127.0.0.1:8090';

export function getMediaUrl(mediaId?: string): string {
  if (!mediaId) return '';
  if (mediaId.startsWith('http://') || mediaId.startsWith('https://')) return mediaId;
  return `${API_BASE}/media/${mediaId}`;
}

export async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T | null> {
  try {
    const url = endpoint.startsWith('http') ? endpoint : `${API_BASE}${endpoint}`;
    const res = await fetch(url, {
      ...options,
      headers: {
        'Accept': 'application/json',
        ...(options?.headers || {}),
      },
      // SSR-friendly cache header
      cache: 'no-store',
    });

    if (!res.ok) {
      console.warn(`API request failed [${res.status}]: ${url}`);
      return null;
    }

    const json = (await res.json()) as ApiResponse<T>;
    return json.data;
  } catch (err) {
    console.error(`API network error on ${endpoint}:`, err);
    return null;
  }
}

export function safeUrl(url?: string): string {
  if (!url) return '#';
  const trimmed = url.trim();
  if (trimmed.startsWith('https://') || trimmed.startsWith('http://') || trimmed.startsWith('/')) {
    return trimmed;
  }
  return '#';
}

export function sanitizeHtml(dirty?: string): string {
  if (!dirty) return '';
  return dirty
    // Remove script tags and contents
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    // Remove style tags and contents
    .replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '')
    // Strip dangerous tags (iframe, object, embed, form, input, base)
    .replace(/<\/?(iframe|object|embed|form|input|button|base|meta|link)\b[^>]*>/gi, '')
    // Strip inline javascript handlers (onerror, onclick, onload, etc.)
    .replace(/\son\w+\s*=\s*(?:'[^']*'|"[^"]*"|[^\s>]+)/gi, '')
    // Strip javascript: pseudo protocol in href or src attributes
    .replace(/(href|src)\s*=\s*['"]\s*javascript:[^'"]*['"]/gi, '$1="#"');
}

export async function getNewsList(params?: {
  category?: string;
  featured?: boolean;
  limit?: number;
  locale?: string;
  search?: string;
}): Promise<NewsArticle[]> {
  const locale = params?.locale || 'tr';
  const query = new URLSearchParams();
  query.set('locale', locale);

  // HydroGo publicList requires locale and supports limit up to 100
  const fetchLimit = params?.category || params?.featured !== undefined ? 100 : (params?.limit || 20);
  query.set('limit', String(Math.min(100, Math.max(1, fetchLimit))));

  const endpoint = `/api/v1/news?${query.toString()}`;
  const data = await fetchApi<NewsArticle[]>(endpoint);
  if (!data) return [];

  let filtered = data;
  if (params?.category) {
    filtered = filtered.filter((item) => item.category === params.category);
  }
  if (params?.featured !== undefined) {
    filtered = filtered.filter((item) => item.featured === params.featured);
  }
  if (params?.search) {
    const s = params.search.toLowerCase();
    filtered = filtered.filter(
      (item) => item.title.toLowerCase().includes(s) || (item.excerpt && item.excerpt.toLowerCase().includes(s))
    );
  }
  if (params?.limit) {
    filtered = filtered.slice(0, params.limit);
  }
  return filtered;
}

export async function getNewsBySlug(slug: string, locale: string = 'tr'): Promise<NewsArticle | null> {
  return await fetchApi<NewsArticle>(`/api/v1/news/${locale}/${slug}`);
}

export async function getColumnists(): Promise<Columnist[]> {
  const data = await fetchApi<Columnist[]>('/api/v1/content/columnists?sort=sort_order');
  return data || [];
}

export async function getAds(zone?: string): Promise<Ad[]> {
  const endpoint = zone ? `/api/v1/content/ads?filter[zone]=${zone}` : '/api/v1/content/ads';
  const data = await fetchApi<Ad[]>(endpoint);
  return data || [];
}

export async function getGalleries(limit: number = 6): Promise<Gallery[]> {
  const data = await fetchApi<Gallery[]>(`/api/v1/content/galleries?limit=${limit}`);
  return data || [];
}

export async function getVideos(limit: number = 6): Promise<VideoNews[]> {
  const data = await fetchApi<VideoNews[]>(`/api/v1/content/videos?limit=${limit}`);
  return data || [];
}

export async function getMasthead(): Promise<Masthead | null> {
  const data = await fetchApi<Masthead[]>('/api/v1/content/masthead?limit=1');
  return data && data.length > 0 ? data[0] : null;
}

export async function getOfficialAnnouncements(limit: number = 20): Promise<OfficialAnnouncement[]> {
  const data = await fetchApi<OfficialAnnouncement[]>(`/api/v1/content/official_announcements?limit=${limit}`);
  return data || [];
}

export async function getActivePoll(): Promise<Poll | null> {
  const data = await fetchApi<Poll[]>('/api/v1/content/polls?limit=1');
  if (data && data.length > 0) return data[0];
  // Fallback demo poll for instant reader interaction
  return {
    id: 'demo-poll-1',
    question: 'Sizce Süper Lig 2026 sezonunda şampiyonluğun en güçlü adayı kim?',
    options: 'Galatasaray,Fenerbahçe,Beşiktaş,Trabzonspor,Diğer',
    total_votes: 1482,
    is_active: true,
  };
}
