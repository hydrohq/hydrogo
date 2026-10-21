import { markDone } from '$lib/server/activities';
import { guardLoad } from '$lib/server/guard';
import { list } from '$lib/server/hydro';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, cookies }) => {
  try {
    const result = await list(locals.session!, 'activities', { limit: 100, sort: 'due', filter: { done: 'false' } });
    return { tasks: result.data };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};

export const actions: Actions = { done: markDone };
