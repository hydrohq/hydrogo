// Optional HTMX adapter for MediaArticle; no HTMX runtime is bundled.
export const MediaArticleHtmx = {
  listTarget: "#media_article-list",
  formTarget: "#media_article-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function MediaArticleHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": MediaArticleHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
