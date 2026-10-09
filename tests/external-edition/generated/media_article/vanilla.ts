// Dependency-free browser adapter for MediaArticle.
export type MediaArticleRequest = { baseUrl: string; fetcher?: typeof fetch };

export function createMediaArticleClient(options: MediaArticleRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/media_article" + query, { headers: { Accept: "application/json" } }) };
}
