// Optional React adapter for MediaArticle. Install React/your query library in the Client Edition.
export type MediaArticleQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type MediaArticleFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type MediaArticleQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function createMediaArticleQuery<T>(fetcher: MediaArticleFetcher<T>, queryKey: readonly unknown[] = ['media_article', 'list']): MediaArticleQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function MediaArticleQueryKey(id?: string): readonly string[] {
  return id ? ['media_article', id] : ['media_article', 'list'];
}
