<script lang="ts">
  import Timeline from '$lib/components/Timeline.svelte';
  import { messages } from '$lib/i18n';
  let { data } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.tasks} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.tasks}</h1>
  <Timeline items={data.tasks} lang={data.lang} />
</main>
