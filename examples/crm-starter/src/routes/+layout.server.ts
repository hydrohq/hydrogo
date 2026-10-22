import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = ({ locals }) => ({ lang: locals.lang, email: locals.session?.email ?? null });
