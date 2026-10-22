import { failure, guardLoad, textFields } from '$lib/server/guard';
import { create, list } from '$lib/server/hydro';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, locals, cookies }) => {
  const search = url.searchParams.get('q') ?? '';
  try {
    const [contacts, companies] = await Promise.all([
      list(locals.session!, 'contacts', { limit: 50, search, expand: ['company'], after: url.searchParams.get('after') ?? undefined }),
      list(locals.session!, 'companies', { limit: 100, sort: 'name' })
    ]);
    return {
      contacts: contacts.data,
      next: contacts.metadata.next_cursor ?? null,
      companies: companies.data.map((c) => ({ id: c.id, name: String(c.name) })),
      search
    };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};

export const actions: Actions = {
  create: async ({ request, locals, cookies }) => {
    const form = await request.formData();
    try {
      await create(locals.session!, 'contacts', { status: 'active', ...textFields(form, ['full_name', 'email', 'phone', 'job_title', 'company']) });
    } catch (error) {
      return failure(error, cookies, locals.sid);
    }
  }
};
