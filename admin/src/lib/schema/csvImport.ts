import { apiRequest, ApiClientError } from '$lib/api/client';

/** The text shown for a failed import: the server's message, the first field errors and the request id. */
export function importErrorMessage(err: unknown, requestIdLabel: string, fallback: string): string {
  if (!(err instanceof ApiClientError)) return fallback;
  const details = Object.entries(err.fields ?? {})
    .slice(0, 3)
    .map(([key, value]) => `${key}: ${value}`)
    .join('; ');
  return `${err.message}${details ? ` — ${details}` : ''}${err.requestId ? ` (${requestIdLabel}: ${err.requestId})` : ''}`;
}

/** Uploads a CSV file to a content type's _import endpoint; returns how many records were created. */
export async function importCsvFile(base: string, file: File): Promise<number> {
  const res = await apiRequest<{ data?: { created?: number } }>(`${base}/_import`, {
    method: 'POST',
    headers: { 'Content-Type': 'text/csv' },
    body: await file.text(),
  });
  return res?.data?.created ?? 0;
}
