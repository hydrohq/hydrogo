import type { RequestEvent } from '@sveltejs/kit';
import { failure, textFields } from './guard';
import { create, patch } from './hydro';

/** Form action: add an activity linked to the given records (deal, company, contact ids). */
export const addActivity = (links: (event: RequestEvent) => Record<string, string>) => async (event: RequestEvent) => {
  const form = await event.request.formData();
  const fields: Record<string, unknown> = { kind: 'task', done: false, ...textFields(form, ['subject', 'kind', 'owner']), ...links(event) };
  const due = String(form.get('due') ?? '');
  if (due) fields.due = new Date(due).toISOString();
  try {
    await create(event.locals.session!, 'activities', fields);
  } catch (error) {
    return failure(error, event.cookies, event.locals.sid);
  }
};

/** Form action: mark an activity done. */
export const markDone = async (event: RequestEvent) => {
  const form = await event.request.formData();
  const id = String(form.get('id') ?? '');
  try {
    await patch(event.locals.session!, 'activities', id, { done: true });
  } catch (error) {
    return failure(error, event.cookies, event.locals.sid);
  }
};
