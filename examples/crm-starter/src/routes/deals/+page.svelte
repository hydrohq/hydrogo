<script lang="ts">
  import { enhance } from '$app/forms';
  import FormError from '$lib/components/FormError.svelte';
  import { messages } from '$lib/i18n';
  import { STAGES, formatMoney } from '$lib/pipeline';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.deals} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.deals}</h1>
  <form method="POST" action="?/create" class="inline" use:enhance>
    <label>{t.name}<input name="name" required /></label>
    <label>{t.company}
      <select name="company" required>{#each data.companies as company}<option value={company.id}>{company.name}</option>{/each}</select>
    </label>
    <label>{t.amount}<input name="amount" type="number" min="0" step="any" /></label>
    <label>{t.currency}<select name="currency"><option>TRY</option><option>USD</option><option>EUR</option></select></label>
    <button class="primary">{t.newDeal}</button>
  </form>
  <FormError {form} lang={data.lang} />
  {#if data.truncated}<p class="muted">100+</p>{/if}
  <div class="board">
    {#each STAGES as stage}
      <section class="column">
        <h3><span>{t.stages[stage]} · {data.columns[stage].length}</span><span class="muted">{formatMoney(data.sums[stage], data.lang)}</span></h3>
        {#each data.columns[stage] as deal}
          <article class="deal">
            <a href={`/deals/${deal.id}`}>{deal.name}</a>
            <div class="muted">{(deal.expanded?.company as { name?: string } | undefined)?.name ?? ''}</div>
            <div>{typeof deal.amount === 'number' ? formatMoney({ [String(deal.currency ?? 'TRY')]: deal.amount }, data.lang) : ''}</div>
            <form method="POST" action="?/move" use:enhance>
              <input type="hidden" name="id" value={deal.id} />
              <select name="stage" aria-label={t.moveTo} onchange={(e) => e.currentTarget.form?.requestSubmit()}>
                {#each STAGES as option}<option value={option} selected={option === stage}>{t.stages[option]}</option>{/each}
              </select>
            </form>
          </article>
        {:else}
          <p class="muted">{t.empty}</p>
        {/each}
      </section>
    {/each}
  </div>
</main>
