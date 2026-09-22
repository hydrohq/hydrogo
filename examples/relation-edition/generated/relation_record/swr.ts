// Optional SWR adapter for RelationRecord. Install SWR in the Client Edition.
export type RelationRecordSWRFetcher<T> = (key: readonly unknown[], signal?: AbortSignal) => Promise<T>;

// This adapter contains no SWR import. Pass the returned key/fetcher to useSWR
// in the client application, keeping HydroGo Core and Compact builds dependency-free.
export function createRelationRecordSWR<T>(fetcher: RelationRecordSWRFetcher<T>, id?: string) {
  const key = id ? ['relation_record', id] as const : ['relation_record', 'list'] as const;
  return { key, fetcher: (signal?: AbortSignal) => fetcher(key, signal), revalidateOnFocus: false };
}
