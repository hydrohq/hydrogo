<script lang="ts">
  import { dealCount, messages } from '$lib/i18n';
  import { formatMoney, isOverdue } from '$lib/pipeline';
  let { data } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.dashboard} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.dashboard}</h1>
  <div class="cards">
    <div class="card"><div class="muted">{t.openPipeline}</div><div class="big">{formatMoney(data.openTotals, data.lang)}</div><div class="muted">{dealCount(t, data.openCount)}</div></div>
    <div class="card"><div class="muted">{t.wonThisPeriod}</div><div class="big">{formatMoney(data.wonTotals, data.lang)}</div><div class="muted">{dealCount(t, data.wonCount)}</div></div>
  </div>
  <h2>{t.dueToday}</h2>
  {#if data.due.length === 0}
    <p class="muted">{t.nothingDue}</p>
  {:else}
    <ul class="timeline">
      {#each data.due as activity}
        <li>
          <span>{activity.subject} <span class="muted">· {t.kinds[activity.kind as keyof typeof t.kinds] ?? activity.kind}</span></span>
          <span class:overdue={isOverdue(activity.due)}>{activity.due ? new Date(activity.due as string).toLocaleString(data.lang) : ''}</span>
        </li>
      {/each}
    </ul>
  {/if}
</main>
