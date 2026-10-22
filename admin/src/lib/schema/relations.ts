import type { Field } from '$lib/api/types';

/** The most relation fields one list request may expand (the server's limit). */
export const MAX_EXPAND = 5;

export function isRelation(field: Field): boolean {
  return field.kind === 'relation';
}

/** True for a relation that holds several records (`many: true` in the YAML). */
export function isManyRelation(field: Field): boolean {
  const relation = field.relation as { cardinality?: string; many?: boolean } | undefined;
  return isRelation(field) && (relation?.cardinality === 'many' || relation?.many === true);
}

export function relationTarget(field: Field): string {
  return field.relation?.collection ?? '';
}

/** The `expand` parameter for the relation fields a list shows, so the rows can name them. */
export function expandParam(fields: Field[]): string {
  return fields
    .filter(isRelation)
    .slice(0, MAX_EXPAND)
    .map((field) => field.id || field.name || '')
    .filter(Boolean)
    .join(',');
}

/** What to call a related record: its type's title field, else its id. */
export function recordTitle(record: Record<string, unknown> | undefined | null, titleField: string | undefined): string {
  if (!record) return '';
  const title = titleField ? record[titleField] : undefined;
  if (title !== undefined && title !== null && String(title).trim() !== '') return String(title);
  return String(record.id ?? '');
}

/**
 * The text for a relation cell in a list. The row carries the plain id(s) and,
 * when the list was expanded, the related records under `expanded`.
 */
export function relationCell(row: Record<string, unknown>, field: Field, titleFieldOf: (collection: string) => string | undefined): string {
  const key = field.id || field.name || '';
  const raw = row[key];
  if (raw === undefined || raw === null || raw === '' || (Array.isArray(raw) && raw.length === 0)) return '—';
  const expanded = (row.expanded as Record<string, unknown> | undefined)?.[key];
  const titleField = titleFieldOf(relationTarget(field));
  if (Array.isArray(expanded)) {
    return expanded.map((record) => recordTitle(record as Record<string, unknown>, titleField)).join(', ') || String(raw);
  }
  if (expanded && typeof expanded === 'object') return recordTitle(expanded as Record<string, unknown>, titleField);
  return Array.isArray(raw) ? raw.join(', ') : String(raw);
}
