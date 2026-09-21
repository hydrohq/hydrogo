// Optional HTMX adapter for Project; no HTMX runtime is bundled.
export const ProjectHtmx = {
  listTarget: "#project-list",
  formTarget: "#project-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function ProjectHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": ProjectHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
