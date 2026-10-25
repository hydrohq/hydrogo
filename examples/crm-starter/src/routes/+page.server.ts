import { guardLoad } from '$lib/server/guard';
import { aggregate, list } from '$lib/server/hydro';
import { OPEN_STAGES, STAGES, isOverdue, winRate } from '$lib/pipeline';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, cookies }) => {
  const session = locals.session!;
  try {
    // Exact figures over every deal: one page of 100 would undercount a busy pipeline.
    const sumsByCurrency = async (stage: string) => {
      const result = await aggregate(session, 'deals', { groupBy: 'currency', sum: ['amount'], filter: { stage } });
      return result.groups.map((group) => ({ currency: group.key ?? 'TRY', value: Number(group.sums.amount ?? 0) }));
    };
    const [byStage, openSums, wonSums, due, recent] = await Promise.all([
      aggregate(session, 'deals', { groupBy: 'stage' }),
      Promise.all(OPEN_STAGES.map(sumsByCurrency)),
      sumsByCurrency('won'),
      list(session, 'activities', { limit: 20, sort: 'due', filter: { done: 'false' } }),
      list(session, 'activities', { limit: 10, sort: '-created_at' })
    ]);
    const stageCounts = Object.fromEntries(STAGES.map((stage) => [stage, 0])) as Record<(typeof STAGES)[number], number>;
    for (const group of byStage.groups) if (group.key && group.key in stageCounts) stageCounts[group.key as (typeof STAGES)[number]] = group.count;
    const sum = (rows: { currency: string; value: number }[]) => {
      const out: Record<string, number> = {};
      for (const row of rows) if (row.value) out[row.currency] = (out[row.currency] ?? 0) + row.value;
      return out;
    };
    return {
      openCount: OPEN_STAGES.reduce((n, stage) => n + stageCounts[stage], 0),
      openTotals: sum(openSums.flat()),
      wonCount: stageCounts.won,
      wonTotals: sum(wonSums),
      due: due.data,
      stageCounts,
      winRate: winRate(stageCounts.won, stageCounts.lost),
      overdueCount: due.data.filter((activity) => isOverdue(activity.due)).length,
      recent: recent.data
    };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};
