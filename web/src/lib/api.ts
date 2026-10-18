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
  votes_data?: string;
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

const PHOTO_MAP: Record<string, string[]> = {
  gundem: [
    'https://images.unsplash.com/photo-1541872703-74c5e44368f9?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1575517111478-7f6afd0973db?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1529107386315-e1a2ed48a620?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1585829365295-ab7cd400c167?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1505373877841-8d25f7d46678?auto=format&fit=crop&w=1200&q=80',
  ],
  ekonomi: [
    'https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1590283603385-17ffb3a7f29f?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1621416894569-0f39ed31d247?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1526304640581-d334cdbbf45e?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1460925895917-afdab827c52f?auto=format&fit=crop&w=1200&q=80',
  ],
  teknoloji: [
    'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1485827404703-89b55fcc595e?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1550751827-4bd374c3f58b?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1517694712202-14dd9538aa97?auto=format&fit=crop&w=1200&q=80',
  ],
  dunya: [
    'https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1477959858617-67f30bc75b82?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1540959733332-eab4deabeeaf?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1513635269975-59663e0ac1ad?auto=format&fit=crop&w=1200&q=80',
  ],
  spor: [
    'https://images.unsplash.com/photo-1508098682722-e99c43a406b2?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1546519638-68e109498ffc?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1461896836934-ffe607ba8211?auto=format&fit=crop&w=1200&q=80',
  ],
  saglik: [
    'https://images.unsplash.com/photo-1532938911079-1b06ac7ceec7?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1505751172876-fa1923c5c528?auto=format&fit=crop&w=1200&q=80',
  ],
  default: [
    'https://images.unsplash.com/photo-1504711434969-e33886168f5c?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1495020689067-958852a7765e?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1497633762265-9d179a990aa6?auto=format&fit=crop&w=1200&q=80',
    'https://images.unsplash.com/photo-1586339949916-3e9457bef6d3?auto=format&fit=crop&w=1200&q=80',
  ],
};

export function getMediaUrl(mediaId?: string, category?: string): string {
  if (mediaId && (mediaId.startsWith('http://') || mediaId.startsWith('https://'))) {
    return mediaId;
  }
  // If demo media ID or no binary media uploaded, return curated editorial photo
  if (!mediaId || mediaId.startsWith('demo-')) {
    const key = mediaId || category || 'haber';
    const cat = (category || '').toLowerCase();
    const photos = PHOTO_MAP[cat] || PHOTO_MAP.default;
    let hash = 0;
    for (let i = 0; i < key.length; i++) {
      hash = (hash << 5) - hash + key.charCodeAt(i);
      hash |= 0;
    }
    const index = Math.abs(hash) % photos.length;
    return photos[index];
  }
  return `${API_BASE}/media/${mediaId}`;
}

export function getCategoryBadge(category?: string): { bg: string; text: string; border: string; glow: string } {
  const cat = (category || '').toLowerCase();
  switch (cat) {
    case 'gundem':
      return { bg: 'bg-rose-600', text: 'text-white', border: 'border-rose-500', glow: 'shadow-rose-600/30' };
    case 'ekonomi':
      return { bg: 'bg-emerald-600', text: 'text-white', border: 'border-emerald-500', glow: 'shadow-emerald-600/30' };
    case 'teknoloji':
      return { bg: 'bg-sky-600', text: 'text-white', border: 'border-sky-500', glow: 'shadow-sky-600/30' };
    case 'spor':
      return { bg: 'bg-amber-600', text: 'text-white', border: 'border-amber-500', glow: 'shadow-amber-600/30' };
    case 'dunya':
      return { bg: 'bg-violet-600', text: 'text-white', border: 'border-violet-500', glow: 'shadow-violet-600/30' };
    case 'saglik':
      return { bg: 'bg-teal-600', text: 'text-white', border: 'border-teal-500', glow: 'shadow-teal-600/30' };
    default:
      return { bg: 'bg-red-600', text: 'text-white', border: 'border-red-500', glow: 'shadow-red-600/30' };
  }
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

/**
 * Converts Quill Delta JSON (or plain text) to semantic HTML.
 * Quill stores content as `{"ops": [{"insert": "..."}, ...]}`.
 */
export function deltaToHtml(raw?: string): string {
  if (!raw) return '';
  const trimmed = raw.trim();

  // If not JSON or does not contain Quill ops, check if plain text needs paragraph wrapping
  if (!trimmed.startsWith('{') || !trimmed.includes('"ops"')) {
    if (/<[a-z][\s\S]*>/i.test(trimmed)) {
      return trimmed;
    }
    return trimmed
      .split(/\n\s*\n/)
      .map(p => `<p class="mb-4 leading-relaxed">${p.trim().replace(/\n/g, '<br/>')}</p>`)
      .join('');
  }

  try {
    const parsed = JSON.parse(trimmed);
    if (!parsed || !Array.isArray(parsed.ops)) {
      return trimmed;
    }

    const ops = parsed.ops;
    let html = '';
    let currentLineBuffer = '';
    let inList: 'bullet' | 'ordered' | null = null;

    const flushLine = (blockAttr?: Record<string, any>) => {
      const content = currentLineBuffer;
      currentLineBuffer = '';

      if (blockAttr?.header) {
        if (inList) {
          html += inList === 'bullet' ? '</ul>' : '</ol>';
          inList = null;
        }
        const level = Math.min(4, Math.max(1, Number(blockAttr.header)));
        const sizeClass = level === 1 ? 'text-2xl sm:text-3xl font-extrabold mt-8 mb-4 text-white border-b border-zinc-800 pb-2' :
                          level === 2 ? 'text-xl sm:text-2xl font-bold mt-6 mb-3 text-white' :
                          'text-lg sm:text-xl font-bold mt-5 mb-2 text-zinc-100';
        html += `<h${level} class="${sizeClass}">${content}</h${level}>`;
      } else if (blockAttr?.blockquote) {
        if (inList) {
          html += inList === 'bullet' ? '</ul>' : '</ol>';
          inList = null;
        }
        html += `<blockquote class="border-l-4 border-red-600 pl-4 py-2 italic my-6 text-zinc-300 bg-zinc-900/50 rounded-r-xl">${content}</blockquote>`;
      } else if (blockAttr?.list) {
        const listType = blockAttr.list === 'ordered' ? 'ordered' : 'bullet';
        if (inList !== listType) {
          if (inList) html += inList === 'bullet' ? '</ul>' : '</ol>';
          html += listType === 'bullet' ? '<ul class="list-disc pl-6 my-4 space-y-1.5 text-zinc-300">' : '<ol class="list-decimal pl-6 my-4 space-y-1.5 text-zinc-300">';
          inList = listType;
        }
        html += `<li>${content}</li>`;
      } else {
        if (inList) {
          html += inList === 'bullet' ? '</ul>' : '</ol>';
          inList = null;
        }
        if (content.trim()) {
          html += `<p class="mb-5 leading-relaxed text-zinc-300">${content}</p>`;
        }
      }
    };

    for (let i = 0; i < ops.length; i++) {
      const op = ops[i];
      if (typeof op.insert === 'object' && op.insert !== null) {
        if (op.insert.image) {
          flushLine();
          html += `<figure class="my-6 rounded-2xl overflow-hidden shadow-2xl bg-zinc-900 border border-zinc-800"><img src="${op.insert.image}" alt="" class="w-full object-cover max-h-[500px]" loading="lazy" /></figure>`;
        }
        continue;
      }

      if (typeof op.insert !== 'string') continue;

      const text = op.insert;
      const lines = text.split('\n');

      for (let j = 0; j < lines.length; j++) {
        const segment = lines[j];

        if (segment.length > 0) {
          let formatted = segment
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;');

          if (op.attributes) {
            if (op.attributes.bold) formatted = `<strong class="font-bold text-white">${formatted}</strong>`;
            if (op.attributes.italic) formatted = `<em>${formatted}</em>`;
            if (op.attributes.underline) formatted = `<u>${formatted}</u>`;
            if (op.attributes.strike) formatted = `<s>${formatted}</s>`;
            if (op.attributes.link) {
              const url = safeUrl(op.attributes.link);
              formatted = `<a href="${url}" target="_blank" rel="noopener noreferrer" class="text-red-400 underline underline-offset-4 hover:text-red-300">${formatted}</a>`;
            }
          }
          currentLineBuffer += formatted;
        }

        if (j < lines.length - 1) {
          flushLine(op.attributes);
        }
      }
    }

    flushLine();
    if (inList) {
      html += inList === 'bullet' ? '</ul>' : '</ol>';
    }

    return html || trimmed;
  } catch (err) {
    console.error('Error parsing Quill Delta:', err);
    return trimmed;
  }
}

export function sanitizeHtml(dirty?: string): string {
  if (!dirty) return '';
  const converted = deltaToHtml(dirty);
  return converted
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    .replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '')
    .replace(/<\/?(iframe|object|embed|form|input|button|base|meta|link)\b[^>]*>/gi, '')
    .replace(/\son\w+\s*=\s*(?:'[^']*'|"[^"]*"|[^\s>]+)/gi, '')
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
    votes_data: JSON.stringify({
      Galatasaray: 620,
      Fenerbahçe: 480,
      Beşiktaş: 230,
      Trabzonspor: 110,
      Diğer: 42,
    }),
    is_active: true,
  };
}
