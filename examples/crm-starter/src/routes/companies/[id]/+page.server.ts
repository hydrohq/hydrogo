import { error } from '@sveltejs/kit';
import { addActivity, markDone } from '$lib/server/activities';
import { failure, guardLoad, textFields } from '$lib/server/guard';
import { HydroError, create, get, list } from '$lib/server/hydro';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, cookies }) => {
  const session = locals.session!;
  try {
    const [company, contacts, deals, activities] = await Promise.all([
      get(session, 'companies', params.id),
      list(session, 'contacts', { limit: 50, filter: { company: params.id } }),
      list(session, 'deals', { limit: 50, filter: { company: params.id } }),
      list(session, 'activities', { limit: 50, sort: '-created_at', filter: { company: params.id } })
    ]);
    return { company, contacts: contacts.data, deals: deals.data, activities: activities.data };
  } catch (e) {
    if (e instanceof HydroError && e.status === 404) error(404, 'Not found');
    guardLoad(e, cookies, locals.sid);
  }
};

export const actions: Actions = {
  activity: addActivity(({ params }) => ({ company: params.id! })),
  done: markDone,
  contact: async ({ request, params, locals, cookies }) => {
    const form = await request.formData();
    try {
      await create(locals.session!, 'contacts', { status: 'active', ...textFields(form, ['full_name', 'email', 'phone', 'job_title']), company: params.id! });
    } catch (e) {
      return failure(e, cookies, locals.sid);
    }
  }
};
