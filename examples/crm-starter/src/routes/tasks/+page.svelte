<script lang="ts">
  import Timeline from '$lib/components/Timeline.svelte';
  import { messages } from '$lib/i18n';
  let { data } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.tasks} · {t.appName}</title></svelte:head>

<div class="page">
  <article class="card"><Timeline items={data.tasks} lang={data.lang} /></article>
</div>
