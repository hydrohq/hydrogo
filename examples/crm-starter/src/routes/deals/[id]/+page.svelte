<script lang="ts">
  import ActivityForm from '$lib/components/ActivityForm.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import FormError from '$lib/components/FormError.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Timeline from '$lib/components/Timeline.svelte';
  import { messages } from '$lib/i18n';
  import { formatMoney } from '$lib/pipeline';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
  const company = $derived(data.deal.expanded?.company as { id: string; name?: string } | undefined);
  const contact = $derived(data.deal.expanded?.contact as { id: string; full_name?: string } | undefined);
</script>

<svelte:head><title>{data.deal.name} · {t.appName}</title></svelte:head>

<div class="page">
  <a class="back" href="/deals"><Icon name="chevron" size={14} /> {t.deals}</a>

  <article class="card profile">
    <Avatar name={String(company?.name ?? data.deal.name)} size="lg" />
    <div class="info">
      <h1>{data.deal.name}</h1>
      <div class="chips"><span class="badge {data.deal.stage}">{t.stages[data.deal.stage as keyof typeof t.stages]}</span></div>
    </div>
    <ActivityForm lang={data.lang} {form} />
  </article>

  <div class="cards">
    <article class="card kpi"><header><span class="icon"><Icon name="deal" size={18} /></span>{t.amount}</header>
      <div class="big money">{typeof data.deal.amount === 'number' ? formatMoney({ [String(data.deal.currency ?? 'TRY')]: data.deal.amount }, data.lang) : '–'}</div></article>
    <article class="card kpi"><header><span class="icon"><Icon name="building" size={18} /></span>{t.company}</header>
      <div class="big money">{#if company}<a href={`/companies/${company.id}`}>{company.name}</a>{:else}–{/if}</div></article>
    <article class="card kpi"><header><span class="icon"><Icon name="users" size={18} /></span>{t.contact}</header>
      <div class="big money">{contact?.full_name ?? '–'}</div></article>
  </div>

  <article class="card">
    <div class="section-head"><h2 class="grow">{t.timeline}</h2></div>
    <Timeline items={data.activities} lang={data.lang} />
  </article>
</div>
