// Optional React adapter for Article. Install React/your query library in the Client Edition.
export type ArticleQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type ArticleFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type ArticleQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function createArticleQuery<T>(fetcher: ArticleFetcher<T>, queryKey: readonly unknown[] = ['article', 'list']): ArticleQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function ArticleQueryKey(id?: string): readonly string[] {
  return id ? ['article', id] : ['article', 'list'];
}
