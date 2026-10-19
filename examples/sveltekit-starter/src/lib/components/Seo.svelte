<script lang="ts">
  import { page } from '$app/state';
  import { localePrefix, locales } from '$lib/config';

  let {
    title,
    description = '',
    path = '/',
    locale,
    image = '',
    type = 'website',
    jsonLd = undefined,
    siteOrigin
  }: {
    title: string;
    description?: string;
    /** Path without the language prefix, e.g. "/news/hello". */
    path?: string;
    locale: string;
    image?: string;
    type?: 'website' | 'article';
    jsonLd?: Record<string, unknown>;
    siteOrigin: string;
  } = $props();

  const url = (code: string) => `${siteOrigin}${localePrefix(code)}${path === '/' ? '' : path}` || `${siteOrigin}/`;
  const canonical = $derived(url(locale));
  // Structured data is serialised on the server; "<" is escaped so content can never close the tag.
  const ld = $derived(jsonLd ? JSON.stringify({ '@context': 'https://schema.org', ...jsonLd }).replaceAll('<', '\\u003c') : '');
  void page;
</script>

<svelte:head>
  <title>{title}</title>
  {#if description}<meta name="description" content={description} />{/if}
  <link rel="canonical" href={canonical} />
  {#if locales.length > 1}
    {#each locales as code (code)}
      <link rel="alternate" hreflang={code} href={url(code)} />
    {/each}
  {/if}
  <meta property="og:type" content={type} />
  <meta property="og:title" content={title} />
  {#if description}<meta property="og:description" content={description} />{/if}
  <meta property="og:url" content={canonical} />
  {#if image}<meta property="og:image" content={image} />{/if}
  <meta name="twitter:card" content={image ? 'summary_large_image' : 'summary'} />
  {#if ld}{@html `<script type="application/ld+json">${ld}</script>`}{/if}
</svelte:head>
