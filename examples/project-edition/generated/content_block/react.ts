// Optional React adapter for ContentBlock. Install React/your query library in the Client Edition.
export type ContentBlockQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type ContentBlockFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type ContentBlockQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function createContentBlockQuery<T>(fetcher: ContentBlockFetcher<T>, queryKey: readonly unknown[] = ['content_block', 'list']): ContentBlockQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function ContentBlockQueryKey(id?: string): readonly string[] {
  return id ? ['content_block', id] : ['content_block', 'list'];
}
