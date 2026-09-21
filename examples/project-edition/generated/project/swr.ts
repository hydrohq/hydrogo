// Optional SWR adapter for Project. Install SWR in the Client Edition.
export type ProjectSWRFetcher<T> = (key: readonly unknown[], signal?: AbortSignal) => Promise<T>;

// This adapter contains no SWR import. Pass the returned key/fetcher to useSWR
// in the client application, keeping HydroGo Core and Compact builds dependency-free.
export function createProjectSWR<T>(fetcher: ProjectSWRFetcher<T>, id?: string) {
  const key = id ? ['project', id] as const : ['project', 'list'] as const;
  return { key, fetcher: (signal?: AbortSignal) => fetcher(key, signal), revalidateOnFocus: false };
}
