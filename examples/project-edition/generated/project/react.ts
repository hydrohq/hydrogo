// Optional React adapter for Project. Install React/your query library in the Client Edition.
export type ProjectQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type ProjectFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type ProjectQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function createProjectQuery<T>(fetcher: ProjectFetcher<T>, queryKey: readonly unknown[] = ['project', 'list']): ProjectQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function ProjectQueryKey(id?: string): readonly string[] {
  return id ? ['project', id] : ['project', 'list'];
}
