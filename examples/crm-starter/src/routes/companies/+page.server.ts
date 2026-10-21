import { redirect } from '@sveltejs/kit';
import { failure, guardLoad, textFields } from '$lib/server/guard';
import { create, list } from '$lib/server/hydro';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, locals, cookies }) => {
  const search = url.searchParams.get('q') ?? '';
  try {
    const result = await list(locals.session!, 'companies', { limit: 50, search, after: url.searchParams.get('after') ?? undefined });
    return { companies: result.data, next: result.metadata.next_cursor ?? null, search };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};

export const actions: Actions = {
  create: async ({ request, locals, cookies }) => {
    const form = await request.formData();
    let id: string;
    try {
      id = (await create(locals.session!, 'companies', textFields(form, ['name', 'industry', 'website', 'phone', 'owner']))).id;
    } catch (error) {
      return failure(error, cookies, locals.sid);
    }
    redirect(303, `/companies/${id}`);
  }
};
