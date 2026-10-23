<script lang="ts">
  import { dealCount, messages } from '$lib/i18n';
  import { STAGES, formatMoney, isOverdue } from '$lib/pipeline';
  let { data } = $props();
  const t = $derived(messages[data.lang]);
  const maxStageCount = $derived(Math.max(1, ...STAGES.map((stage) => data.stageCounts[stage])));
</script>

<svelte:head><title>{t.dashboard} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.dashboard}</h1>
  <div class="cards">
    <div class="card"><div class="muted">{t.openPipeline}</div><div class="big">{formatMoney(data.openTotals, data.lang)}</div><div class="muted">{dealCount(t, data.openCount)}</div></div>
    <div class="card"><div class="muted">{t.wonThisPeriod}</div><div class="big">{formatMoney(data.wonTotals, data.lang)}</div><div class="muted">{dealCount(t, data.wonCount)}</div></div>
    <div class="card">
      <div class="muted">{t.winRate}</div>
      <div class="big">{data.winRate === null ? '–' : new Intl.NumberFormat(data.lang, { style: 'percent', maximumFractionDigits: 0 }).format(data.winRate)}</div>
      <div class="muted">{data.winRate === null ? t.noClosedDeals : ''}</div>
    </div>
    <div class="card"><div class="muted">{t.overdueCount}</div><div class="big" class:overdue={data.overdueCount > 0}>{data.overdueCount}</div></div>
  </div>

  <h2>{t.stageBreakdown}</h2>
  <div class="bars">
    {#each STAGES as stage}
      <div class="bar-row">
        <span class="bar-label">{t.stages[stage]}</span>
        <div class="bar-track"><div class="bar-fill" style="width:{(data.stageCounts[stage] / maxStageCount) * 100}%"></div></div>
        <span class="muted">{data.stageCounts[stage]}</span>
      </div>
    {/each}
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

  <h2>{t.recentActivity}</h2>
  {#if data.recent.length === 0}
    <p class="muted">{t.empty}</p>
  {:else}
    <ul class="timeline">
      {#each data.recent as activity}
        <li>
          <span>{activity.subject} <span class="muted">· {t.kinds[activity.kind as keyof typeof t.kinds] ?? activity.kind}</span></span>
          <span class="muted">{new Date(activity.created_at).toLocaleString(data.lang)}</span>
        </li>
      {/each}
    </ul>
  {/if}
</main>
