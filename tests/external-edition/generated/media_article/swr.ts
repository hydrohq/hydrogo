// Optional SWR adapter for MediaArticle. Install SWR in the Client Edition.
export type MediaArticleSWRFetcher<T> = (key: readonly unknown[], signal?: AbortSignal) => Promise<T>;

// This adapter contains no SWR import. Pass the returned key/fetcher to useSWR
// in the client application, keeping HydroGo Core and Compact builds dependency-free.
export function createMediaArticleSWR<T>(fetcher: MediaArticleSWRFetcher<T>, id?: string) {
  const key = id ? ['media_article', id] as const : ['media_article', 'list'] as const;
  return { key, fetcher: (signal?: AbortSignal) => fetcher(key, signal), revalidateOnFocus: false };
}
