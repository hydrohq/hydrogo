// Dependency-free browser adapter for ContentBlock.
export type ContentBlockRequest = { baseUrl: string; fetcher?: typeof fetch };

export function createContentBlockClient(options: ContentBlockRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/content_block" + query, { headers: { Accept: "application/json" } }) };
}
