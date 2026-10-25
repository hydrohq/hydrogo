<script lang="ts">
  import { dealCount, messages } from '$lib/i18n';
  import { OPEN_STAGES, STAGES, formatMoney, isOverdue } from '$lib/pipeline';
  import BarChart from '$lib/components/BarChart.svelte';
  import CountUp from '$lib/components/CountUp.svelte';
  import DonutChart from '$lib/components/DonutChart.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { fly } from 'svelte/transition';
  import { ease, ms } from '$lib/motion';
  let { data } = $props();
  const t = $derived(messages[data.lang]);
  const stageLabels = $derived(STAGES.map((s) => t.stages[s]));
  const stageValues = $derived(STAGES.map((s) => data.stageCounts[s]));
  const colors = ['var(--s1)', 'var(--s2)', 'var(--s3)', 'var(--s4)'];
  const mix = $derived(OPEN_STAGES.map((s, i) => ({ label: t.stages[s], value: data.stageCounts[s], color: colors[i] })));
  const rate = $derived(data.winRate === null ? null : Math.round(data.winRate * 100));
</script>

<svelte:head><title>{t.dashboard} · {t.appName}</title></svelte:head>

<div class="page">
  <div class="cards">
    <article class="card kpi">
      <header><span class="icon"><Icon name="deal" size={18} /></span>{t.openPipeline}<a class="go" href="/deals" aria-label={t.deals}><Icon name="arrow" size={15} /></a></header>
      <div class="big money">{formatMoney(data.openTotals, data.lang)}</div>
      <span class="muted">{dealCount(t, data.openCount)}</span>
    </article>
    <article class="card kpi">
      <header><span class="icon"><Icon name="check" size={18} /></span>{t.wonThisPeriod}<a class="go" href="/deals" aria-label={t.deals}><Icon name="arrow" size={15} /></a></header>
      <div class="big money">{formatMoney(data.wonTotals, data.lang)}</div>
      <span class="muted">{dealCount(t, data.wonCount)}</span>
    </article>
    <article class="card kpi">
      <header><span class="icon"><Icon name="chart" size={18} /></span>{t.winRate}</header>
      <div class="big">{#if rate === null}–{:else}<CountUp value={rate} suffix="%" />{/if}</div>
      <span class="muted">{rate === null ? t.noClosedDeals : ' '}</span>
    </article>
    <article class="card kpi">
      <header><span class="icon"><Icon name="clock" size={18} /></span>{t.overdueCount}<a class="go" href="/tasks" aria-label={t.tasks}><Icon name="arrow" size={15} /></a></header>
      <div class="big" class:overdue={data.overdueCount > 0}><CountUp value={data.overdueCount} /></div>
      <span class="muted">{data.overdueCount > 0 ? t.dueToday : t.nothingDue}</span>
    </article>
  </div>

  <div class="split">
    <article class="card">
      <div class="section-head"><h2 class="grow">{t.stageBreakdown}</h2></div>
      <BarChart labels={stageLabels} values={stageValues} highlight={stageValues.indexOf(Math.max(...stageValues))} unit={t.dealMany} />
    </article>
    <article class="card">
      <div class="section-head"><h2 class="grow">{t.pipelineMix}</h2></div>
      <DonutChart slices={mix} centerLabel={t.openCount} />
    </article>
  </div>

  <div class="split even">
    <article class="card">
      <div class="section-head"><h2 class="grow">{t.dueToday}</h2><a class="see-all" href="/tasks">{t.viewAll}</a></div>
      {#if data.due.length === 0}
        <p class="muted">{t.nothingDue}</p>
      {:else}
        <div class="rowlist">
          {#each data.due.slice(0, 6) as activity, i (activity.id)}
            <div class="rowcard" in:fly|local={{ y: 10, duration: ms(260), delay: ms(i * 40), easing: ease }}>
              <span class="avatar" style="background:var(--surface-2);color:var(--accent)"><Icon name={activity.kind === 'call' ? 'phone' : activity.kind === 'email' ? 'mail' : activity.kind === 'meeting' ? 'calendar' : activity.kind === 'note' ? 'note' : 'tasks'} size={16} /></span>
              <div class="main"><span class="title">{activity.subject}</span><div class="sub">{t.kinds[activity.kind as keyof typeof t.kinds] ?? activity.kind}</div></div>
              <span class="trailing" class:overdue={isOverdue(activity.due)}>{activity.due ? new Date(activity.due as string).toLocaleString(data.lang, { dateStyle: 'short', timeStyle: 'short' }) : ''}</span>
            </div>
          {/each}
        </div>
      {/if}
    </article>
    <article class="card">
      <div class="section-head"><h2 class="grow">{t.recentActivity}</h2></div>
      {#if data.recent.length === 0}
        <p class="muted">{t.empty}</p>
      {:else}
        <div class="rowlist">
          {#each data.recent.slice(0, 6) as activity, i (activity.id)}
            <div class="rowcard" in:fly|local={{ y: 10, duration: ms(260), delay: ms(i * 40), easing: ease }}>
              <span class="avatar" style="background:var(--surface-2);color:var(--accent)"><Icon name={activity.kind === 'call' ? 'phone' : activity.kind === 'email' ? 'mail' : activity.kind === 'meeting' ? 'calendar' : activity.kind === 'note' ? 'note' : 'tasks'} size={16} /></span>
              <div class="main"><span class="title">{activity.subject}</span><div class="sub">{t.kinds[activity.kind as keyof typeof t.kinds] ?? activity.kind}</div></div>
              <span class="trailing muted">{new Date(activity.created_at).toLocaleString(data.lang, { dateStyle: 'short', timeStyle: 'short' })}</span>
            </div>
          {/each}
        </div>
      {/if}
    </article>
  </div>
</div>
