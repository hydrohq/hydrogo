import { guardLoad } from '$lib/server/guard';
import { list } from '$lib/server/hydro';
import { OPEN_STAGES, STAGES, isOverdue, totals, winRate } from '$lib/pipeline';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, cookies }) => {
  const session = locals.session!;
  try {
    const [open, won, lost, due, recent] = await Promise.all([
      Promise.all(OPEN_STAGES.map((stage) => list(session, 'deals', { limit: 100, filter: { stage } }))),
      list(session, 'deals', { limit: 100, filter: { stage: 'won' } }),
      list(session, 'deals', { limit: 100, filter: { stage: 'lost' } }),
      list(session, 'activities', { limit: 20, sort: 'due', filter: { done: 'false' } }),
      list(session, 'activities', { limit: 10, sort: '-created_at' })
    ]);
    const openDeals = open.flatMap((result) => result.data);
    const stageCounts = Object.fromEntries(STAGES.map((stage) => [stage, 0])) as Record<(typeof STAGES)[number], number>;
    OPEN_STAGES.forEach((stage, i) => (stageCounts[stage] = open[i].data.length));
    stageCounts.won = won.data.length;
    stageCounts.lost = lost.data.length;
    return {
      openCount: openDeals.length,
      openTotals: totals(openDeals),
      wonCount: won.data.length,
      wonTotals: totals(won.data),
      due: due.data,
      stageCounts,
      winRate: winRate(won.data.length, lost.data.length),
      overdueCount: due.data.filter((activity) => isOverdue(activity.due)).length,
      recent: recent.data
    };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};
