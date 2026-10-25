<script lang="ts">
  import { enhance } from '$app/forms';
  import type { SubmitFunction } from '@sveltejs/kit';
  import ActivityForm from '$lib/components/ActivityForm.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import Empty from '$lib/components/Empty.svelte';
  import FormError from '$lib/components/FormError.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Timeline from '$lib/components/Timeline.svelte';
  import { messages } from '$lib/i18n';
  import { formatMoney } from '$lib/pipeline';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);

  let open = $state(false);
  const submit: SubmitFunction = () => async ({ result, update }) => {
    await update();
    if (result.type === 'success') open = false;
  };
</script>

<svelte:head><title>{data.company.name} · {t.appName}</title></svelte:head>

<div class="page">
  <a class="back" href="/companies"><Icon name="chevron" size={14} /> {t.companies}</a>

  <article class="card profile">
    <Avatar name={String(data.company.name)} size="lg" />
    <div class="info">
      <h1>{data.company.name}</h1>
      <div class="chips">
        {#each [data.company.industry, data.company.website, data.company.phone].filter(Boolean) as chip (chip)}<span class="chip">{chip}</span>{/each}
      </div>
    </div>
    <ActivityForm lang={data.lang} {form} />
  </article>
  {#if !open}<FormError {form} lang={data.lang} />{/if}

  <div class="split even">
    <article class="card">
      <div class="section-head"><h2 class="grow">{t.contacts}</h2><button class="add" onclick={() => (open = true)}><Icon name="plus" size={16} /> {t.newContact}</button></div>
      {#if data.contacts.length === 0}
        <p class="muted">{t.empty}</p>
      {:else}
        <div class="rowlist">
          {#each data.contacts as contact (contact.id)}
            <div class="rowcard">
              <Avatar name={String(contact.full_name)} />
              <div class="main"><span class="title">{contact.full_name}</span><div class="sub">{contact.job_title ?? ''}</div></div>
              <span class="trailing muted">{contact.email ?? ''}</span>
            </div>
          {/each}
        </div>
      {/if}
    </article>
    <article class="card">
      <div class="section-head"><h2 class="grow">{t.deals}</h2></div>
      {#if data.deals.length === 0}
        <p class="muted">{t.empty}</p>
      {:else}
        <div class="rowlist">
          {#each data.deals as deal (deal.id)}
            <div class="rowcard">
              <div class="main"><a class="title" href={`/deals/${deal.id}`}>{deal.name}</a></div>
              <span class="trailing"><span class="badge {deal.stage}">{t.stages[deal.stage as keyof typeof t.stages]}</span>{typeof deal.amount === 'number' ? formatMoney({ [String(deal.currency ?? 'TRY')]: deal.amount }, data.lang) : ''}</span>
            </div>
          {/each}
        </div>
      {/if}
    </article>
  </div>

  <article class="card">
    <div class="section-head"><h2 class="grow">{t.timeline}</h2></div>
    <Timeline items={data.activities} lang={data.lang} />
  </article>
</div>

{#if open}
  <Modal title={t.newContact} closeLabel={t.close} onclose={() => (open = false)}>
    <form id="contact-form" method="POST" action="?/contact" use:enhance={submit} class="formgrid">
      <label class="wide">{t.name}<input name="full_name" required /></label>
      <label>{t.email}<input name="email" type="email" /></label>
      <label>{t.phone}<input name="phone" /></label>
      <label class="wide">{t.title}<input name="job_title" /></label>
      {#if form?.error}<div class="wide"><FormError {form} lang={data.lang} /></div>{/if}
    </form>
    {#snippet footer()}
      <button type="button" onclick={() => (open = false)}>{t.cancel}</button>
      <button class="primary" form="contact-form">{t.save}</button>
    {/snippet}
  </Modal>
{/if}
