<script lang="ts">
  import type { NewsItem } from '$lib/hydrogo';

  let { item, href, cover = '' }: { item: NewsItem; href: string; cover?: string } = $props();
  const date = $derived(new Date(item.created_at ?? item.updated_at).toLocaleDateString(undefined, { dateStyle: 'medium' }));
</script>

<article class="card">
  {#if cover}<a href={href} tabindex="-1" aria-hidden="true"><img src={cover} alt="" loading="lazy" /></a>{/if}
  <div>
    {#if item.category}<p class="eyebrow">{item.category}</p>{/if}
    <h2><a {href}>{item.title}</a></h2>
    {#if item.excerpt}<p>{item.excerpt}</p>{/if}
    <p class="meta"><time datetime={item.created_at ?? item.updated_at}>{date}</time>{#if item.author_name} · {item.author_name}{/if}</p>
  </div>
</article>
