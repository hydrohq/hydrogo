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
    <div class="card">
      <div class="icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2 3 7v10l9 5 9-5V7l-9-5Z" /></svg></div>
      <div class="muted">{t.openPipeline}</div>
      <div class="big">{formatMoney(data.openTotals, data.lang)}</div>
      <div class="muted">{dealCount(t, data.openCount)}</div>
    </div>
    <div class="card">
      <div class="icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg></div>
      <div class="muted">{t.wonThisPeriod}</div>
      <div class="big">{formatMoney(data.wonTotals, data.lang)}</div>
      <div class="muted">{dealCount(t, data.wonCount)}</div>
    </div>
    <div class="card">
      <div class="icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18M7 14l4-4 4 4 5-6" /></svg></div>
      <div class="muted">{t.winRate}</div>
      <div class="big">{data.winRate === null ? '–' : new Intl.NumberFormat(data.lang, { style: 'percent', maximumFractionDigits: 0 }).format(data.winRate)}</div>
      <div class="muted">{data.winRate === null ? t.noClosedDeals : ''}</div>
    </div>
    <div class="card">
      <div class="icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 3" /></svg></div>
      <div class="muted">{t.overdueCount}</div>
      <div class="big" class:overdue={data.overdueCount > 0}>{data.overdueCount}</div>
    </div>
  </div>

  <div class="section-card" style="margin-top: 1.4rem;">
    <div class="section-head"><strong>{t.stageBreakdown}</strong></div>
    <div class="bars">
      {#each STAGES as stage}
        <div class="bar-row">
          <span class="muted">{t.stages[stage]}</span>
          <div class="bar-track"><div class="bar-fill" style="width:{(data.stageCounts[stage] / maxStageCount) * 100}%"></div></div>
          <span class="muted">{data.stageCounts[stage]}</span>
        </div>
      {/each}
    </div>
  </div>

  <div class="section-card" style="margin-top: .9rem;">
    <div class="section-head"><strong>{t.dueToday}</strong></div>
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
  </div>

  <div class="section-card" style="margin-top: .9rem;">
    <div class="section-head"><strong>{t.recentActivity}</strong></div>
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
  </div>
</main>
