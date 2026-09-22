// Dependency-free browser adapter for Article.
export type ArticleRequest = { baseUrl: string; fetcher?: typeof fetch };

export function createArticleClient(options: ArticleRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/article" + query, { headers: { Accept: "application/json" } }) };
}
