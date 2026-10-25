export type Dir = 1 | -1;

/** A sortable value: numbers compare as numbers, everything else as locale-aware text. */
export function sortRows<T>(rows: T[], value: (row: T) => unknown, dir: Dir, lang: string): T[] {
  const key = (row: T) => {
    const v = value(row);
    return typeof v === 'number' ? v : String(v ?? '').toLocaleLowerCase(lang);
  };
  return [...rows].sort((a, b) => {
    const x = key(a);
    const y = key(b);
    const order = typeof x === 'number' && typeof y === 'number' ? x - y : String(x).localeCompare(String(y), lang, { numeric: true });
    return order * dir;
  });
}

/** Clicking the same column flips the direction; a new column starts ascending. */
export function nextSort(current: { key: string; dir: Dir }, key: string): { key: string; dir: Dir } {
  return current.key === key ? { key, dir: (current.dir === 1 ? -1 : 1) as Dir } : { key, dir: 1 };
}
