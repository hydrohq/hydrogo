// Optional React adapter for RelationRecord. Install React/your query library in the Client Edition.
export type RelationRecordQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type RelationRecordFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type RelationRecordQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function createRelationRecordQuery<T>(fetcher: RelationRecordFetcher<T>, queryKey: readonly unknown[] = ['relation_record', 'list']): RelationRecordQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function RelationRecordQueryKey(id?: string): readonly string[] {
  return id ? ['relation_record', id] : ['relation_record', 'list'];
}
