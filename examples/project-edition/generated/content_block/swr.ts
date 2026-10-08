// Optional SWR adapter for ContentBlock. Install SWR in the Client Edition.
export type ContentBlockSWRFetcher<T> = (key: readonly unknown[], signal?: AbortSignal) => Promise<T>;

// This adapter contains no SWR import. Pass the returned key/fetcher to useSWR
// in the client application, keeping HydroGo Core and Compact builds dependency-free.
export function createContentBlockSWR<T>(fetcher: ContentBlockSWRFetcher<T>, id?: string) {
  const key = id ? ['content_block', id] as const : ['content_block', 'list'] as const;
  return { key, fetcher: (signal?: AbortSignal) => fetcher(key, signal), revalidateOnFocus: false };
}
