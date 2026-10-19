import type { NewsArticle, Page } from './api';

export interface SEOOptions {
  title?: string;
  description?: string;
  keywords?: string[];
  canonical?: string;
  image?: string;
  imageAlt?: string;
  type?: 'website' | 'article';
  publishedTime?: string;
  author?: string;
  jsonLd?: Record<string, any>;
}

const DEFAULT_ORIGIN = typeof window !== 'undefined' ? window.location.origin : 'https://example.com';
const DEFAULT_TITLE = 'HydroGo · Headless CMS in a single Go binary';
const DEFAULT_DESCRIPTION =
  'HydroGo is a single-binary headless CMS written in Go: pages, news, media, members and YAML content types behind one REST API, with an embedded admin panel.';
const DEFAULT_KEYWORDS = [
  'HydroGo',
  'headless CMS',
  'Golang CMS',
  'Svelte 5',
  'REST API',
  'zero-build publishing',
  'high performance CMS',
  'developer-first CMS',
];
const DEFAULT_OG_IMAGE = `${DEFAULT_ORIGIN}/og-image.png`;

function setMetaTag(attrName: 'name' | 'property', attrValue: string, content: string | undefined) {
  if (typeof document === 'undefined') return;
  let element = document.querySelector(`meta[${attrName}="${attrValue}"]`) as HTMLMetaElement | null;
  if (!content) {
    if (element) element.remove();
    return;
  }
  if (!element) {
    element = document.createElement('meta');
    element.setAttribute(attrName, attrValue);
    document.head.appendChild(element);
  }
  element.setAttribute('content', content);
}

function setCanonical(url: string | undefined) {
  if (typeof document === 'undefined') return;
  let link = document.querySelector('link[rel="canonical"]') as HTMLLinkElement | null;
  if (!url) {
    if (link) link.remove();
    return;
  }
  if (!link) {
    link = document.createElement('link');
    link.setAttribute('rel', 'canonical');
    document.head.appendChild(link);
  }
  link.setAttribute('href', url);
}

function setJsonLd(schema: Record<string, any> | undefined) {
  if (typeof document === 'undefined') return;
  const SCRIPT_ID = 'hydrogo-jsonld-schema';
  let script = document.getElementById(SCRIPT_ID) as HTMLScriptElement | null;
  if (!schema) {
    if (script) script.remove();
    return;
  }
  if (!script) {
    script = document.createElement('script');
    script.id = SCRIPT_ID;
    script.type = 'application/ld+json';
    document.head.appendChild(script);
  }
  script.textContent = JSON.stringify(schema, null, 2);
}

export function setSEO(options: SEOOptions = {}) {
  if (typeof document === 'undefined') return;

  const title = options.title ? `${options.title} · HydroGo` : DEFAULT_TITLE;
  const description = options.description || DEFAULT_DESCRIPTION;
  const keywords = options.keywords && options.keywords.length > 0 ? options.keywords.join(', ') : DEFAULT_KEYWORDS.join(', ');
  const canonical = options.canonical || (typeof window !== 'undefined' ? window.location.href : DEFAULT_ORIGIN);
  const image = options.image || DEFAULT_OG_IMAGE;
  const type = options.type || 'website';

  // Browser Document Title
  document.title = title;

  // Standard Meta Tags
  setMetaTag('name', 'description', description);
  setMetaTag('name', 'keywords', keywords);
  setMetaTag('name', 'robots', 'index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1');
  if (options.author) {
    setMetaTag('name', 'author', options.author);
  }

  // Canonical Link
  setCanonical(canonical);

  // Open Graph / Facebook / WhatsApp / LinkedIn
  setMetaTag('property', 'og:site_name', 'HydroGo');
  setMetaTag('property', 'og:type', type);
  setMetaTag('property', 'og:title', title);
  setMetaTag('property', 'og:description', description);
  setMetaTag('property', 'og:url', canonical);
  setMetaTag('property', 'og:image', image);
  setMetaTag('property', 'og:image:width', '1200');
  setMetaTag('property', 'og:image:height', '630');
  setMetaTag('property', 'og:image:type', 'image/png');
  if (options.imageAlt) {
    setMetaTag('property', 'og:image:alt', options.imageAlt);
  }

  // Twitter Cards
  setMetaTag('name', 'twitter:card', 'summary_large_image');
  setMetaTag('name', 'twitter:title', title);
  setMetaTag('name', 'twitter:description', description);
  setMetaTag('name', 'twitter:image', image);

  // JSON-LD Structured Data
  if (options.jsonLd) {
    setJsonLd(options.jsonLd);
  } else {
    setJsonLd({
      '@context': 'https://schema.org',
      '@type': 'WebSite',
      name: 'HydroGo',
      url: DEFAULT_ORIGIN,
      description: DEFAULT_DESCRIPTION,
      publisher: {
        '@type': 'Organization',
        name: 'HydroGo',
        url: DEFAULT_ORIGIN,
      },
    });
  }
}

export function setDefaultSEO() {
  const origin = typeof window !== 'undefined' ? window.location.origin : DEFAULT_ORIGIN;
  setSEO({
    title: '', // Reverts to DEFAULT_TITLE
    description: DEFAULT_DESCRIPTION,
    canonical: origin + '/',
    type: 'website',
    jsonLd: {
      '@context': 'https://schema.org',
      '@type': 'WebSite',
      name: 'HydroGo',
      url: origin + '/',
      description: DEFAULT_DESCRIPTION,
      publisher: {
        '@type': 'Organization',
        name: 'HydroGo',
        url: origin,
      },
    },
  });
}

export function setDocsSEO(pageTitle?: string, description?: string) {
  const origin = typeof window !== 'undefined' ? window.location.origin : DEFAULT_ORIGIN;
  const url = typeof window !== 'undefined' ? window.location.href : `${origin}/#/docs`;
  setSEO({
    title: pageTitle ? `${pageTitle} · Docs` : 'Documentation',
    description: description || 'Developer documentation for HydroGo, a single-binary headless CMS written in Go.',
    canonical: url,
    type: 'website',
    jsonLd: {
      '@context': 'https://schema.org',
      '@type': 'TechArticle',
      headline: pageTitle || 'HydroGo documentation',
      description: description || 'Developer documentation for HydroGo.',
      url,
      inLanguage: 'en',
    },
  });
}

export function setArticleSEO(article: NewsArticle) {
  const origin = typeof window !== 'undefined' ? window.location.origin : DEFAULT_ORIGIN;
  const title = article.seo_title || article.title;
  const description = article.seo_description || article.excerpt || article.title;
  const image = article.cover_media_id ? `${origin}/media/${article.cover_media_id}` : DEFAULT_OG_IMAGE;
  const canonical = `${origin}/#news?slug=${encodeURIComponent(article.slug)}`;

  setSEO({
    title,
    description,
    canonical,
    image,
    imageAlt: article.title,
    keywords: article.tags,
    type: 'article',
    author: article.author_name || 'HydroGo',
    publishedTime: article.published_at,
    jsonLd: {
      '@context': 'https://schema.org',
      '@type': 'NewsArticle',
      headline: title,
      description,
      image: [image],
      datePublished: article.published_at,
      dateModified: article.published_at,
      author: {
        '@type': 'Person',
        name: article.author_name || 'HydroGo Editorial',
      },
      publisher: {
        '@type': 'Organization',
        name: 'HydroGo',
        url: origin,
      },
      mainEntityOfPage: {
        '@type': 'WebPage',
        '@id': canonical,
      },
    },
  });
}

export function setPageSEO(page: Page) {
  const origin = typeof window !== 'undefined' ? window.location.origin : DEFAULT_ORIGIN;
  const canonical = `${origin}/#page?slug=${encodeURIComponent(page.slug)}`;
  const title = page.title;
  const description = `Read ${page.title} on HydroGo.`;

  setSEO({
    title,
    description,
    canonical,
    type: 'website',
    jsonLd: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: title,
      description,
      url: canonical,
      publisher: {
        '@type': 'Organization',
        name: 'HydroGo',
        url: origin,
      },
    },
  });
}
