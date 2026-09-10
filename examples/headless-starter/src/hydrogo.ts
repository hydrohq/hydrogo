export type Block = { type: string; payload: Record<string, unknown> };
export type Page = { id: string; locale: string; slug: string; title: string; body: string; blocks: Block[] };

const origin = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env?.HYDRO_API_ORIGIN ?? "http://127.0.0.1:8080";

export async function getPage(locale: string, slug: string): Promise<Page> {
  const response = await fetch(`${origin}/api/v1/pages/by-slug?locale=${encodeURIComponent(locale)}&slug=${encodeURIComponent(slug)}`);
  if (!response.ok) throw new Error(`HydroGo returned ${response.status}`);
  return response.json() as Promise<Page>;
}
