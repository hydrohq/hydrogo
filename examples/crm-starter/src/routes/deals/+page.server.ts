import { fail } from '@sveltejs/kit';
import { failure, guardLoad, textFields } from '$lib/server/guard';
import { create, list, patch } from '$lib/server/hydro';
import { STAGES, byStage, totals } from '$lib/pipeline';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, cookies }) => {
  const session = locals.session!;
  try {
    const [deals, companies] = await Promise.all([
      list(session, 'deals', { limit: 100, sort: '-updated_at', expand: ['company'] }),
      list(session, 'companies', { limit: 100, sort: 'name' })
    ]);
    const columns = byStage(deals.data);
    return {
      columns: Object.fromEntries(STAGES.map((stage) => [stage, columns[stage]])),
      sums: Object.fromEntries(STAGES.map((stage) => [stage, totals(columns[stage])])),
      companies: companies.data.map((c) => ({ id: c.id, name: String(c.name) })),
      truncated: deals.metadata.has_more
    };
  } catch (error) {
    guardLoad(error, cookies, locals.sid);
  }
};

export const actions: Actions = {
  move: async ({ request, locals, cookies }) => {
    const form = await request.formData();
    const id = String(form.get('id') ?? '');
    const stage = String(form.get('stage') ?? '');
    if (!id || !(STAGES as readonly string[]).includes(stage)) return fail(400, { error: 'api', message: 'invalid stage' });
    try {
      await patch(locals.session!, 'deals', id, { stage });
    } catch (error) {
      return failure(error, cookies, locals.sid);
    }
  },
  create: async ({ request, locals, cookies }) => {
    const form = await request.formData();
    const fields: Record<string, unknown> = { stage: 'lead', ...textFields(form, ['name', 'company', 'currency', 'owner']) };
    const amount = Number(form.get('amount'));
    if (form.get('amount') && Number.isFinite(amount)) fields.amount = amount;
    try {
      await create(locals.session!, 'deals', fields);
    } catch (error) {
      return failure(error, cookies, locals.sid);
    }
  }
};
