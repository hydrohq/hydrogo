// Optional React adapter for Milestone. Install React/your query library in the Client Edition.
export type MilestoneQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type MilestoneFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type MilestoneQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function createMilestoneQuery<T>(fetcher: MilestoneFetcher<T>, queryKey: readonly unknown[] = ['milestone', 'list']): MilestoneQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function MilestoneQueryKey(id?: string): readonly string[] {
  return id ? ['milestone', id] : ['milestone', 'list'];
}
