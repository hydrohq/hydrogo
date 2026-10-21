<script lang="ts">
  import ActivityForm from '$lib/components/ActivityForm.svelte';
  import FormError from '$lib/components/FormError.svelte';
  import Timeline from '$lib/components/Timeline.svelte';
  import { messages } from '$lib/i18n';
  import { formatMoney } from '$lib/pipeline';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
  const company = $derived(data.deal.expanded?.company as { id: string; name?: string } | undefined);
  const contact = $derived(data.deal.expanded?.contact as { id: string; full_name?: string } | undefined);
</script>

<svelte:head><title>{data.deal.name} · {t.appName}</title></svelte:head>

<main class="page">
  <p><a href="/deals">← {t.deals}</a></p>
  <h1>{data.deal.name}</h1>
  <div class="cards">
    <div class="card"><div class="muted">{t.stage}</div><div class="big">{t.stages[data.deal.stage as keyof typeof t.stages]}</div></div>
    <div class="card"><div class="muted">{t.amount}</div><div class="big">{typeof data.deal.amount === 'number' ? formatMoney({ [String(data.deal.currency ?? 'TRY')]: data.deal.amount }, data.lang) : '–'}</div></div>
    <div class="card"><div class="muted">{t.company}</div>{#if company}<a href={`/companies/${company.id}`}>{company.name}</a>{/if}<div class="muted">{contact?.full_name ?? ''}</div></div>
  </div>
  <h2>{t.timeline}</h2>
  <ActivityForm lang={data.lang} />
  <FormError {form} lang={data.lang} />
  <Timeline items={data.activities} lang={data.lang} />
</main>
