import { error } from '@sveltejs/kit';
import { addActivity, markDone } from '$lib/server/activities';
import { guardLoad } from '$lib/server/guard';
import { HydroError, get, list } from '$lib/server/hydro';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, cookies }) => {
  try {
    const [deal, activities] = await Promise.all([
      get(locals.session!, 'deals', params.id, ['company', 'contact']),
      list(locals.session!, 'activities', { limit: 50, sort: '-created_at', filter: { deal: params.id } })
    ]);
    return { deal, activities: activities.data };
  } catch (e) {
    if (e instanceof HydroError && e.status === 404) error(404, 'Not found');
    guardLoad(e, cookies, locals.sid);
  }
};

export const actions: Actions = {
  activity: addActivity(({ params }) => ({ deal: params.id! })),
  done: markDone
};
