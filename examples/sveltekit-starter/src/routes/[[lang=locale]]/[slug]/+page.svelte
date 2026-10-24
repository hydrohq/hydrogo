<script lang="ts">
  import Seo from '$lib/components/Seo.svelte';
  import { sanitizeHtml } from '$lib/sanitizeHtml';

  let { data } = $props();
</script>

<Seo title="{data.page.title} · {data.siteName}" path="/{data.page.slug}" locale={data.locale} siteOrigin={data.siteOrigin} />

<article class="article">
  <h1>{data.page.title}</h1>
  <div class="prose">{@html sanitizeHtml(data.page.body)}</div>
</article>
