// SvelteKit loads param matchers with a plain Node import, outside the
// Vite pipeline, so no $env module (static or dynamic) is resolvable
// here; read the same variable directly from process.env instead.
//
// /tr/... is a language prefix only when "tr" is one of PUBLIC_LOCALES;
// otherwise the segment is an ordinary page slug.
export function match(param: string): boolean {
  const configured = (process.env.PUBLIC_LOCALES ?? 'en').split(',').map((v) => v.trim());
  return configured.slice(1).includes(param);
}
