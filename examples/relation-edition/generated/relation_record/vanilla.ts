// Dependency-free browser adapter for RelationRecord.
export type RelationRecordRequest = { baseUrl: string; fetcher?: typeof fetch };

export function createRelationRecordClient(options: RelationRecordRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/relation_record" + query, { headers: { Accept: "application/json" } }) };
}
