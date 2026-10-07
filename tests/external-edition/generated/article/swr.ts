// Optional SWR adapter for Article. Install SWR in the Client Edition.
export type ArticleSWRFetcher<T> = (key: readonly unknown[], signal?: AbortSignal) => Promise<T>;

// This adapter contains no SWR import. Pass the returned key/fetcher to useSWR
// in the client application, keeping HydroGo Core and Compact builds dependency-free.
export function createArticleSWR<T>(fetcher: ArticleSWRFetcher<T>, id?: string) {
  const key = id ? ['article', id] as const : ['article', 'list'] as const;
  return { key, fetcher: (signal?: AbortSignal) => fetcher(key, signal), revalidateOnFocus: false };
}
