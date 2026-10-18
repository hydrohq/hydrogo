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

export interface PageData {
  id: string;
  locale: string;
  slug: string;
  title: string;
  body: string;
  tags?: string[];
  status: string;
  created_at: string;
  updated_at: string;
}

export interface NavItem {
  id: string;
  label: string;
  href: string;
  placement: 'header' | 'footer_sayfalar' | 'footer_sosyal';
  sort_order?: number;
}

export interface SiteSettings {
  id?: string;
  site_name: string;
  tagline?: string;
  footer_description?: string;
  footer_highlight_title?: string;
  footer_highlight_text?: string;
  footer_highlight_cta_text?: string;
  footer_highlight_cta_href?: string;
  copyright_text?: string;
  seo_title?: string;
  seo_description?: string;
  favicon?: string;
  comments_enabled?: boolean;
  comments_require_email?: boolean;
}

export type HomepageSectionKey =
  | 'hero'
  | 'son_haberler'
  | 'bulten'
  | 'ikinci_grid'
  | 'cok_okunan'
  | 'galeri'
  | 'kategori_vitrini';

export interface HomepageSection {
  id: string;
  key: HomepageSectionKey;
  category_filter?: string;
  visible: boolean;
  sort_order: number;
}

const FALLBACK_HOMEPAGE_SECTIONS: HomepageSection[] = (
  ['hero', 'son_haberler', 'bulten', 'ikinci_grid', 'cok_okunan', 'galeri'] as HomepageSectionKey[]
).map((key, i) => ({ id: `fallback-${key}`, key, visible: true, sort_order: i + 1 }));

export async function getHomepageSections(): Promise<HomepageSection[]> {
  const data = await fetchApi<HomepageSection[]>('/api/v1/content/homepage_sections?limit=50&sort=sort_order');
  const sections = data && data.length > 0 ? data : FALLBACK_HOMEPAGE_SECTIONS;
  return [...sections].sort((a, b) => a.sort_order - b.sort_order).filter((s) => s.visible);
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

export type AdPageScope = 'tumu' | 'anasayfa' | 'kategori' | 'haber_detay';

export interface Ad {
  id: string;
  name: string;
  zone: string;
  ad_type?: 'banner' | 'native' | 'script';
  page_scope?: AdPageScope;
  disabled?: boolean;
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

export interface Comment {
  id: string;
  article_slug: string;
  author_name: string;
  body: string;
  created_at: string;
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

// Self-hosted under static/images/editorial so the starter never depends on
// a live third-party image host (hotlinked Unsplash URLs go dead over time —
// two of the originals already had, which is how this got noticed).
const PHOTO_MAP: Record<string, string[]> = {
  gundem: ['/images/editorial/gundem-1.jpg', '/images/editorial/gundem-2.jpg', '/images/editorial/gundem-3.jpg', '/images/editorial/gundem-4.jpg', '/images/editorial/gundem-5.jpg'],
  ekonomi: ['/images/editorial/ekonomi-1.jpg', '/images/editorial/ekonomi-2.jpg', '/images/editorial/ekonomi-3.jpg', '/images/editorial/ekonomi-4.jpg', '/images/editorial/ekonomi-5.jpg'],
  teknoloji: ['/images/editorial/teknoloji-1.jpg', '/images/editorial/teknoloji-2.jpg', '/images/editorial/teknoloji-3.jpg', '/images/editorial/teknoloji-4.jpg', '/images/editorial/teknoloji-5.jpg'],
  dunya: ['/images/editorial/dunya-1.jpg', '/images/editorial/dunya-2.jpg', '/images/editorial/dunya-3.jpg', '/images/editorial/dunya-4.jpg'],
  spor: ['/images/editorial/spor-1.jpg', '/images/editorial/spor-2.jpg', '/images/editorial/spor-3.jpg', '/images/editorial/spor-4.jpg', '/images/editorial/spor-5.jpg', '/images/editorial/spor-6.jpg', '/images/editorial/spor-7.jpg'],
  saglik: ['/images/editorial/saglik-1.jpg', '/images/editorial/saglik-2.jpg'],
  default: ['/images/editorial/default-1.jpg', '/images/editorial/default-2.jpg', '/images/editorial/default-3.jpg', '/images/editorial/default-4.jpg'],
};

export function slugify(text?: string): string {
  if (!text) return 'haber';
  return text
    .toString()
    .toLowerCase()
    .trim()
    .replace(/ğ/g, 'g')
    .replace(/ü/g, 'u')
    .replace(/ş/g, 's')
    .replace(/ı/g, 'i')
    .replace(/ö/g, 'o')
    .replace(/ç/g, 'c')
    .replace(/&/g, 've')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || 'haber';
}

export function getMediaUrl(mediaId?: string, category?: string, seed?: string, index?: number): string {
  if (mediaId && (mediaId.startsWith('http://') || mediaId.startsWith('https://'))) {
    return mediaId;
  }
  // If demo media ID or no binary media uploaded, return curated editorial photo
  if (!mediaId || mediaId.startsWith('demo-')) {
    const cat = slugify(category);
    let photos = PHOTO_MAP.default;

    if (cat.includes('ekonomi') || cat.includes('ticaret') || cat.includes('finans')) {
      photos = PHOTO_MAP.ekonomi;
    } else if (cat.includes('teknoloji') || cat.includes('yazilim') || cat.includes('muhendislik') || cat.includes('urun') || cat.includes('tasarim') || cat.includes('performans') || cat.includes('veritabani') || cat.includes('guvenlik')) {
      photos = PHOTO_MAP.teknoloji;
    } else if (cat.includes('spor')) {
      photos = PHOTO_MAP.spor;
    } else if (cat.includes('dunya') || cat.includes('rehber')) {
      photos = PHOTO_MAP.dunya;
    } else if (cat.includes('saglik') || cat.includes('eczane')) {
      photos = PHOTO_MAP.saglik;
    } else if (cat.includes('gundem') || cat.includes('siyaset') || cat.includes('haber')) {
      photos = PHOTO_MAP.gundem;
    }

    // When the caller knows its position in a rendered list (`index`), use
    // it directly — guarantees no two cards in the same grid repeat a photo
    // as long as the pool is at least as large as the list. Otherwise (a
    // lone article, e.g. on its own detail page) fall back to hashing the
    // slug so the same article always gets the same photo across renders.
    if (index !== undefined) {
      return photos[index % photos.length];
    }
    const key = seed || mediaId || category || 'haber';
    let hash = 0;
    for (let i = 0; i < key.length; i++) {
      hash = (hash << 5) - hash + key.charCodeAt(i);
      hash |= 0;
    }
    return photos[Math.abs(hash) % photos.length];
  }
  return `${API_BASE}/media/${mediaId}`;
}

// Plain media URL with no category-photo fallback — for chrome like a
// favicon where a random stock photo standing in would be wrong.
export function getAssetUrl(mediaId?: string): string | null {
  if (!mediaId) return null;
  if (mediaId.startsWith('http://') || mediaId.startsWith('https://')) return mediaId;
  return `${API_BASE}/media/${mediaId}`;
}

export function getCategoryBadge(category?: string): { bg: string; text: string; border: string; glow: string } {
  const cat = slugify(category);

  // Soft pastel pill per beat — light tint background, saturated text, no
  // border/shadow. A small, deliberate set so tags read as one system.
  if (cat.includes('ekonomi') || cat.includes('ticaret') || cat.includes('finans')) {
    return { bg: 'bg-amber-100', text: 'text-amber-800', border: 'border-transparent', glow: '' };
  }
  if (cat.includes('spor')) {
    return { bg: 'bg-orange-100', text: 'text-orange-800', border: 'border-transparent', glow: '' };
  }
  if (
    cat.includes('teknoloji') ||
    cat.includes('yazilim') ||
    cat.includes('muhendislik') ||
    cat.includes('urun') ||
    cat.includes('veritabani') ||
    cat.includes('performans') ||
    cat.includes('tasarim') ||
    cat.includes('frontend') ||
    cat.includes('bilim')
  ) {
    return { bg: 'bg-violet-100', text: 'text-violet-800', border: 'border-transparent', glow: '' };
  }
  if (cat.includes('magazin') || cat.includes('kultur') || cat.includes('sanat')) {
    return { bg: 'bg-pink-100', text: 'text-pink-800', border: 'border-transparent', glow: '' };
  }
  if (cat.includes('dunya') || cat.includes('amerika') || cat.includes('rehber')) {
    return { bg: 'bg-blue-100', text: 'text-blue-800', border: 'border-transparent', glow: '' };
  }
  if (cat.includes('saglik') || cat.includes('eczane')) {
    return { bg: 'bg-teal-100', text: 'text-teal-800', border: 'border-transparent', glow: '' };
  }
  if (cat.includes('guvenlik') || cat.includes('mimari')) {
    return { bg: 'bg-slate-100', text: 'text-slate-800', border: 'border-transparent', glow: '' };
  }
  // Default: gündem, siyaset, haber and anything unclassified.
  return { bg: 'bg-red-100', text: 'text-red-800', border: 'border-transparent', glow: '' };
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

export async function getPage(slug: string, locale: string = 'tr'): Promise<PageData | null> {
  return await fetchApi<PageData>(`/api/v1/pages/${locale}/${slug}`);
}

// Fallback shown only if the panel has no site_navigation / site_settings
// records yet (e.g. a fresh install before `seed-site-content.mjs` runs) —
// the admin panel is the real source of truth once seeded.
const FALLBACK_NAV: NavItem[] = [
  { id: 'fallback-son-dakika', label: 'Son Dakika', href: '/son-dakika', placement: 'header', sort_order: 0 },
  { id: 'fallback-gundem', label: 'Gündem', href: '/gundem', placement: 'header', sort_order: 1 },
  { id: 'fallback-dunya', label: 'Dünya', href: '/dunya', placement: 'header', sort_order: 2 },
  { id: 'fallback-ekonomi', label: 'Ekonomi', href: '/ekonomi', placement: 'header', sort_order: 3 },
  { id: 'fallback-teknoloji', label: 'Teknoloji', href: '/teknoloji', placement: 'header', sort_order: 4 },
  { id: 'fallback-spor', label: 'Spor', href: '/spor', placement: 'header', sort_order: 5 },
  { id: 'fallback-kunye', label: 'Künye', href: '/kunye', placement: 'header', sort_order: 6 },
  { id: 'fallback-anasayfa', label: 'Anasayfa', href: '/', placement: 'footer_sayfalar', sort_order: 1 },
  { id: 'fallback-haberler', label: 'Haberler', href: '/gundem', placement: 'footer_sayfalar', sort_order: 2 },
];

const FALLBACK_SETTINGS: SiteSettings = {
  site_name: 'HydroPost',
  tagline: 'Bağımsız Haber Portalı',
  footer_description: 'HydroGo motoru üzerinde çalışan bağımsız, ultra hızlı ve güvenilir haber portalı.',
  footer_highlight_title: 'HydroPost',
  footer_highlight_text: 'Türkiye ve dünya gündemini bir araya getiriyor.',
  footer_highlight_cta_text: 'Haberleri Oku',
  footer_highlight_cta_href: '/gundem',
  copyright_text: 'HydroPost Haber Portalı. Tüm hakları saklıdır.',
  seo_title: 'HydroPost — Bağımsız ve Güçlü Haber Portalı',
  seo_description: 'HydroGo tekil binary mimarisiyle çalışan, ışık hızında haber portalı.',
};

export async function getSiteNavigation(placement?: NavItem['placement']): Promise<NavItem[]> {
  const endpoint = `/api/v1/content/site_navigation?limit=100&sort=sort_order${placement ? `&filter[placement]=${placement}` : ''}`;
  const data = await fetchApi<NavItem[]>(endpoint);
  if (data && data.length > 0) return data;
  return placement ? FALLBACK_NAV.filter((n) => n.placement === placement) : FALLBACK_NAV;
}

export async function getSiteSettings(): Promise<SiteSettings> {
  const data = await fetchApi<SiteSettings[]>('/api/v1/content/site_settings?limit=1');
  return data && data.length > 0 ? data[0] : FALLBACK_SETTINGS;
}

export async function getColumnists(): Promise<Columnist[]> {
  const data = await fetchApi<Columnist[]>('/api/v1/content/columnists?sort=sort_order');
  return data || [];
}

export async function getAds(zone?: string, pageScope?: AdPageScope): Promise<Ad[]> {
  const endpoint = zone ? `/api/v1/content/ads?limit=50&filter[zone]=${zone}` : '/api/v1/content/ads?limit=50';
  const data = await fetchApi<Ad[]>(endpoint);
  // `disabled` and `page_scope` aren't server-filterable (booleans/selects
  // can't be combined with another filter in one query in HydroGo's
  // content-type API), so both the site-wide kill switch and the per-page
  // scope are applied here.
  return (data || []).filter((ad) => !ad.disabled && (!pageScope || !ad.page_scope || ad.page_scope === 'tumu' || ad.page_scope === pageScope));
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

export async function getComments(slug: string): Promise<Comment[]> {
  const data = await fetchApi<Comment[]>(`/api/v1/content/comments?limit=50&filter[article_slug]=${encodeURIComponent(slug)}`);
  return data || [];
}

export async function getActivePoll(): Promise<Poll | null> {
  // Respect the editor's "Aktif mi?" toggle — only ever surface a poll that
  // was explicitly turned on. Boolean fields aren't filterable server-side
  // in HydroGo's content-type query API, so this filters client-side.
  const data = await fetchApi<Poll[]>('/api/v1/content/polls?limit=20');
  const active = data?.find((p) => p.is_active);
  if (active) return active;
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
