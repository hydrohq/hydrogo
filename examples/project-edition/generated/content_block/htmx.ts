// Optional HTMX adapter for ContentBlock; no HTMX runtime is bundled.
export const ContentBlockHtmx = {
  listTarget: "#content_block-list",
  formTarget: "#content_block-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function ContentBlockHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": ContentBlockHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
