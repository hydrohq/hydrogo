<script lang="ts">
  import { enhance } from '$app/forms';
  import type { SubmitFunction } from '@sveltejs/kit';
  import { fly } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import Avatar from '$lib/components/Avatar.svelte';
  import FormError from '$lib/components/FormError.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import { messages } from '$lib/i18n';
  import { ease, ms } from '$lib/motion';
  import { STAGES, formatMoney } from '$lib/pipeline';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
  const dot: Record<string, string> = { lead: 'var(--s1)', qualified: 'var(--s2)', proposal: 'var(--s3)', negotiation: 'var(--s4)', won: 'var(--s6)', lost: 'var(--s8)' };
  let open = $state(false);
  const submit: SubmitFunction = () => async ({ result, update }) => {
    await update();
    if (result.type === 'success') open = false;
  };
</script>

<svelte:head><title>{t.deals} · {t.appName}</title></svelte:head>

<div class="page">
  <div class="toolbar-row" style="display:flex;justify-content:flex-end">
    <button class="primary add" onclick={() => (open = true)}><Icon name="plus" size={16} /> {t.newDeal}</button>
  </div>
  {#if !open}<FormError {form} lang={data.lang} />{/if}
  {#if data.truncated}<p class="muted">100+</p>{/if}
  <div class="board">
    {#each STAGES as stage (stage)}
      <section class="column">
        <h3><span class="dot" style="background:{dot[stage]}"></span>{t.stages[stage]}<span class="pill">{data.columns[stage].length}</span><span class="sum">{formatMoney(data.sums[stage], data.lang)}</span></h3>
        {#each data.columns[stage] as deal (deal.id)}
          {@const company = (deal.expanded?.company as { name?: string } | undefined)?.name ?? ''}
          <article class="deal" in:fly|local={{ y: 12, duration: ms(280), easing: ease }} animate:flip={{ duration: ms(260) }}>
            <div class="top">
              {#if company}<Avatar name={company} />{/if}
              <div style="min-width:0"><a href={`/deals/${deal.id}`}>{deal.name}</a><div class="muted small">{company}</div></div>
            </div>
            <div class="amount">{typeof deal.amount === 'number' ? formatMoney({ [String(deal.currency ?? 'TRY')]: deal.amount }, data.lang) : ''}</div>
            <form method="POST" action="?/move" use:enhance>
              <input type="hidden" name="id" value={deal.id} />
              <select name="stage" aria-label={t.moveTo} onchange={(e) => e.currentTarget.form?.requestSubmit()}>
                {#each STAGES as option (option)}<option value={option} selected={option === stage}>{t.stages[option]}</option>{/each}
              </select>
            </form>
          </article>
        {:else}
          <p class="muted" style="padding:.4rem .5rem">{t.empty}</p>
        {/each}
      </section>
    {/each}
  </div>
</div>

{#if open}
  <Modal title={t.newDeal} closeLabel={t.close} onclose={() => (open = false)}>
    <form id="deal-form" method="POST" action="?/create" use:enhance={submit} class="formgrid">
      <label class="wide">{t.name}<input name="name" required /></label>
      <label class="wide">{t.company}
        <select name="company" required>{#each data.companies as company (company.id)}<option value={company.id}>{company.name}</option>{/each}</select>
      </label>
      <label>{t.amount}<input name="amount" type="number" min="0" step="any" /></label>
      <label>{t.currency}<select name="currency"><option>TRY</option><option>USD</option><option>EUR</option></select></label>
      {#if form?.error}<div class="wide"><FormError {form} lang={data.lang} /></div>{/if}
    </form>
    {#snippet footer()}
      <button type="button" onclick={() => (open = false)}>{t.cancel}</button>
      <button class="primary" form="deal-form">{t.save}</button>
    {/snippet}
  </Modal>
{/if}
