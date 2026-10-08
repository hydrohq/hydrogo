// Optional HTMX adapter for Article; no HTMX runtime is bundled.
export const ArticleHtmx = {
  listTarget: "#article-list",
  formTarget: "#article-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function ArticleHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": ArticleHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
