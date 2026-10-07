// Dependency-free browser adapter for Milestone.
export type MilestoneRequest = { baseUrl: string; fetcher?: typeof fetch };

export function createMilestoneClient(options: MilestoneRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/milestone" + query, { headers: { Accept: "application/json" } }) };
}
