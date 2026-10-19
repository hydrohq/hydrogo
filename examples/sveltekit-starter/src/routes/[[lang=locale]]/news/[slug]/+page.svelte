<script lang="ts">
  import Seo from '$lib/components/Seo.svelte';

  let { data } = $props();
  const item = $derived(data.item);
  const title = $derived(item.seo_title || item.title);
  const description = $derived(item.seo_description || item.excerpt || '');
</script>

<Seo
  title="{title} · {data.siteName}"
  {description}
  path="/news/{item.slug}"
  locale={data.locale}
  image={data.cover}
  type="article"
  siteOrigin={data.siteOrigin}
  jsonLd={{
    '@type': 'NewsArticle',
    headline: item.title,
    description,
    image: data.cover || undefined,
    datePublished: item.created_at,
    dateModified: item.updated_at,
    author: item.author_name ? { '@type': 'Person', name: item.author_name } : undefined
  }}
/>

<article class="article">
  {#if item.category}<p class="eyebrow">{item.category}</p>{/if}
  <h1>{item.title}</h1>
  <p class="meta">{#if item.author_name}{item.author_name} · {/if}<time datetime={item.updated_at}>{new Date(item.updated_at).toLocaleDateString(undefined, { dateStyle: 'long' })}</time></p>
  {#if data.cover}<img class="cover" src={data.cover} alt="" />{/if}
  <!-- HydroGo sanitizes rich text on the server, so it is safe to render as HTML. -->
  <div class="prose">{@html item.body}</div>
</article>
