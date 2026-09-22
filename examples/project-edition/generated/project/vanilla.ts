// Dependency-free browser adapter for Project.
export type ProjectRequest = { baseUrl: string; fetcher?: typeof fetch };

export function createProjectClient(options: ProjectRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/project" + query, { headers: { Accept: "application/json" } }) };
}
