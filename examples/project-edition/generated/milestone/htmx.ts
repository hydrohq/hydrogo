// Optional HTMX adapter for Milestone; no HTMX runtime is bundled.
export const MilestoneHtmx = {
  listTarget: "#milestone-list",
  formTarget: "#milestone-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function MilestoneHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": MilestoneHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
