import { guardLoad } from '$lib/server/guard';
import { list } from '$lib/server/hydro';
import { OPEN_STAGES, totals } from '$lib/pipeline';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, cookies }) => {
  const session = locals.session!;
  try {
    const [open, won, due] = await Promise.all([
      Promise.all(OPEN_STAGES.map((stage) => list(session, 'deals', { limit: 100, filter: { stage } }))),
      list(session, 'deals', { limit: 100, filter: { stage: 'won' } }),
      list(session, 'activities', { limit: 20, sort: 'due', filter: { done: 'false' } })
    ]);
    const openDeals = open.flatMap((result) => result.data);
    return {
      openCount: openDeals.length,
      openTotals: totals(openDeals),
      wonCount: won.data.length,
      wonTotals: totals(won.data),
      due: due.data
    };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};
