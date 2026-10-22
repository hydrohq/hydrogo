<script lang="ts">
  import { messages, type Lang } from '$lib/i18n';
  let { form, lang }: { form: { error?: string; message?: string; requestId?: string; fields?: Record<string, string> } | null | undefined; lang: Lang } = $props();
  const t = $derived(messages[lang]);
</script>

{#if form?.error}
  <p class="error" role="alert">
    {form.error === 'conflict' ? t.conflict : form.message || t.error}
    {#each Object.entries(form.fields ?? {}) as [field, problem]}<span> {field}: {problem}.</span>{/each}
    {#if form.requestId}<small class="muted"> {t.requestId}: {form.requestId}</small>{/if}
  </p>
{/if}
