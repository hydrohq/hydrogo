<script lang="ts">
  import { enhance } from '$app/forms';
  import ActivityForm from '$lib/components/ActivityForm.svelte';
  import FormError from '$lib/components/FormError.svelte';
  import Timeline from '$lib/components/Timeline.svelte';
  import { messages } from '$lib/i18n';
  import { formatMoney } from '$lib/pipeline';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{data.company.name} · {t.appName}</title></svelte:head>

<main class="page">
  <p><a href="/companies">← {t.companies}</a></p>
  <h1>{data.company.name}</h1>
  <p class="muted">{[data.company.industry, data.company.website, data.company.phone].filter(Boolean).join(' · ')}</p>
  <FormError {form} lang={data.lang} />

  <h2>{t.contacts}</h2>
  <form method="POST" action="?/contact" class="inline" use:enhance>
    <label>{t.name}<input name="full_name" required /></label>
    <label>{t.email}<input name="email" type="email" /></label>
    <label>{t.phone}<input name="phone" /></label>
    <label>{t.title}<input name="job_title" /></label>
    <button class="primary">{t.newContact}</button>
  </form>
  <ul class="timeline">
    {#each data.contacts as contact}<li><span>{contact.full_name} <span class="muted">{contact.job_title ?? ''}</span></span><span class="muted">{contact.email ?? ''}</span></li>{:else}<li class="muted">{t.empty}</li>{/each}
  </ul>

  <h2>{t.deals}</h2>
  <ul class="timeline">
    {#each data.deals as deal}
      <li><a href={`/deals/${deal.id}`}>{deal.name}</a><span>{t.stages[deal.stage as keyof typeof t.stages]} · {typeof deal.amount === 'number' ? formatMoney({ [String(deal.currency ?? 'TRY')]: deal.amount }, data.lang) : ''}</span></li>
    {:else}<li class="muted">{t.empty}</li>{/each}
  </ul>

  <h2>{t.timeline}</h2>
  <ActivityForm lang={data.lang} />
  <Timeline items={data.activities} lang={data.lang} />
</main>
