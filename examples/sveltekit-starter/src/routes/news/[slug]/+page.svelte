<script lang="ts">
  let { data } = $props();
  const item = $derived(data.item);
  const canonical = $derived(data.canonical);
  const ld = $derived(
    JSON.stringify({
      '@context': 'https://schema.org',
      '@type': 'NewsArticle',
      headline: item.title,
      dateModified: item.updated_at,
      mainEntityOfPage: canonical
    }).replaceAll('<', '\\u003c')
  );
</script>

<!-- Rendered on the server, so crawlers and link previews see all of this. -->
<svelte:head>
  <title>{item.title}</title>
  <link rel="canonical" href={canonical} />
  <meta property="og:type" content="article" />
  <meta property="og:title" content={item.title} />
  <meta property="og:url" content={canonical} />
  {@html `<script type="application/ld+json">${ld}</script>`}
</svelte:head>

<article>
  <h1>{item.title}</h1>
  <!-- The API sanitizes rich text server-side (bluemonday), so it is safe to render. -->
  {@html item.body}
</article>
