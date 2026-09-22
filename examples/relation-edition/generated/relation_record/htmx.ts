// Optional HTMX adapter for RelationRecord; no HTMX runtime is bundled.
export const RelationRecordHtmx = {
  listTarget: "#relation_record-list",
  formTarget: "#relation_record-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function RelationRecordHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": RelationRecordHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
