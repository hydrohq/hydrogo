import { api } from '$lib/api/client';

// The list API rejects a filter value longer than 256 bytes (collections.MaxQueryValueBytes),
// so a long id list for filter[product][in]= must be split.
const MAX_FILTER_BYTES = 240;
const PAGE = 100;

/** Splits ids into groups whose comma-joined, URL-encoded value stays under maxBytes. */
export function chunkIdsByBytes(ids: string[], maxBytes = MAX_FILTER_BYTES): string[][] {
  const chunks: string[][] = [];
  let current: string[] = [];
  let size = 0;
  for (const id of ids) {
    const len = encodeURIComponent(id).length + 1;
    if (current.length > 0 && size + len > maxBytes) {
      chunks.push(current);
      current = [];
      size = 0;
    }
    current.push(id);
    size += len;
  }
  if (current.length > 0) chunks.push(current);
  return chunks;
}

/** All variants of the given products: id chunks are fetched in parallel, each chunk paged until a short page. */
export async function fetchVariantsForProducts<T>(productIds: string[]): Promise<T[]> {
  const chunks = chunkIdsByBytes(productIds.filter(Boolean));
  const perChunk = await Promise.all(
    chunks.map(async (chunk) => {
      const inClause = chunk.map(encodeURIComponent).join(',');
      const rows: T[] = [];
      for (let offset = 0; ; offset += PAGE) {
        const res = await api.get<{ data: T[]; metadata?: { has_more?: boolean } }>(
          `/api/v1/admin/content/variants?limit=${PAGE}&offset=${offset}&filter[product][in]=${inClause}`
        );
        const batch = res.data ?? [];
        rows.push(...batch);
        if (batch.length < PAGE) break;
      }
      return rows;
    })
  );
  return perChunk.flat();
}
