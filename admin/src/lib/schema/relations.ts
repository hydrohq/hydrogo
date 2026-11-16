import type { Field } from '$lib/api/types';

/** The most relation fields one list request may expand (the server's limit). */
export const MAX_EXPAND = 5;

export function isRelation(field: Field): boolean {
  return field.kind === 'relation' || field.widget === 'relation';
}

/** True for a relation that holds several records (`many: true` in the YAML). */
export function isManyRelation(field: Field): boolean {
  const relation = field.relation as { cardinality?: string; many?: boolean } | undefined;
  return isRelation(field) && (relation?.cardinality === 'many' || relation?.many === true);
}

export function relationTarget(field: Field): string {
  return field.relation?.collection ?? '';
}

/** Which field on the target record is stored (`id` for native relations, `slug` or custom for text relations). */
export function relationValueField(field: Field): string {
  return field.relation?.value_field || (field.kind === 'relation' ? 'id' : 'slug');
}

/** The `expand` parameter for native relation fields a list shows, so the rows can name them. */
export function expandParam(fields: Field[]): string {
  return fields
    .filter((field) => field.kind === 'relation')
    .slice(0, MAX_EXPAND)
    .map((field) => field.id || field.name || '')
    .filter(Boolean)
    .join(',');
}

/** Extracts selected keys from either an array, a comma-separated string, or a single string. */
export function parseRelationKeys(raw: unknown, valueField: string, many: boolean): string[] {
  const seen = new Set<string>();
  const add = (val: string, out: string[]) => {
    const clean = val.trim();
    if (clean && !seen.has(clean)) {
      seen.add(clean);
      out.push(clean);
    }
  };
  const out: string[] = [];
  if (Array.isArray(raw)) {
    for (const item of raw) {
      if (item !== undefined && item !== null) add(String(item), out);
    }
    return out;
  }
  if (typeof raw === 'string') {
    const trimmed = raw.trim();
    if (!trimmed) return [];
    if (many || (valueField !== 'id' && many)) {
      for (const part of trimmed.split(',')) add(part, out);
      return out;
    }
    return [trimmed];
  }
  return [];
}

/** Formats selected keys back into the field's storage representation. */
export function serializeRelationKeys(keys: string[], valueField: string, many: boolean): string | string[] {
  if (valueField === 'id') {
    return many ? keys : (keys[0] ?? '');
  }
  return many ? keys.join(', ') : (keys[0] ?? '');
}

/** Resolves a display thumbnail URL from common media properties on a related record. */
export function resolveRecordMedia(record: Record<string, unknown> | undefined | null): string {
  if (!record) return '';
  for (const key of ['cover_image', 'image', 'poster', 'thumbnail'] as const) {
    const raw = record[key];
    if (typeof raw !== 'string') continue;
    const trimmed = raw.trim();
    if (!trimmed) continue;
    if (trimmed.startsWith('http://') || trimmed.startsWith('https://') || trimmed.startsWith('/') || trimmed.startsWith('data:')) {
      return trimmed;
    }
    return `/media/${trimmed}`;
  }
  return '';
}

/** What to call a related record: its type's title field, else its id. */
export function recordTitle(record: Record<string, unknown> | undefined | null, titleField: string | undefined): string {
  if (!record) return '';
  const title = titleField ? record[titleField] : undefined;
  if (title !== undefined && title !== null && String(title).trim() !== '') return String(title);
  return String(record.id ?? '');
}

/**
 * The text for a relation cell in a list. The row carries the plain key(s) and,
 * when the list was expanded (or a client lookup map exists), the related titles.
 */
export function relationCell(
  row: Record<string, unknown>,
  field: Field,
  titleFieldOf: (collection: string) => string | undefined,
  lookupTitle?: (target: string, key: string) => string | undefined
): string {
  const key = field.id || field.name || '';
  const raw = row[key];
  if (raw === undefined || raw === null || raw === '' || (Array.isArray(raw) && raw.length === 0)) return '—';
  const expanded = (row.expanded as Record<string, unknown> | undefined)?.[key];
  const target = relationTarget(field);
  const titleField = titleFieldOf(target);
  if (Array.isArray(expanded)) {
    return expanded.map((record) => recordTitle(record as Record<string, unknown>, titleField)).join(', ') || String(raw);
  }
  if (expanded && typeof expanded === 'object') return recordTitle(expanded as Record<string, unknown>, titleField);
  if (lookupTitle) {
    const keys = parseRelationKeys(raw, relationValueField(field), isManyRelation(field));
    if (keys.length > 0) {
      return keys.map((k) => lookupTitle(target, k) || k).join(', ');
    }
  }
  return Array.isArray(raw) ? raw.join(', ') : String(raw);
}
