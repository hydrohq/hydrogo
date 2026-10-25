import { describe, expect, it } from 'vitest';
import { nextSort, sortRows } from './table';

describe('sortRows', () => {
  const rows = [{ n: 'Çelik' }, { n: 'Arı' }, { n: 'Zeynep' }, { n: 'Ece' }];
  it('sorts text with the locale rules, ascending and descending', () => {
    expect(sortRows(rows, (r) => r.n, 1, 'tr').map((r) => r.n)).toEqual(['Arı', 'Çelik', 'Ece', 'Zeynep']);
    expect(sortRows(rows, (r) => r.n, -1, 'tr').map((r) => r.n)).toEqual(['Zeynep', 'Ece', 'Çelik', 'Arı']);
  });
  it('sorts numbers as numbers and keeps missing values together', () => {
    const r = [{ v: 10 }, { v: 9 }, { v: undefined }, { v: 100 }];
    expect(sortRows(r, (x) => x.v, 1, 'en').map((x) => x.v)).toEqual([undefined, 9, 10, 100]);
  });
  it('does not mutate its input', () => {
    const copy = [...rows];
    sortRows(rows, (r) => r.n, 1, 'en');
    expect(rows).toEqual(copy);
  });
});

describe('nextSort', () => {
  it('flips on the same column and restarts on another', () => {
    expect(nextSort({ key: 'name', dir: 1 }, 'name')).toEqual({ key: 'name', dir: -1 });
    expect(nextSort({ key: 'name', dir: -1 }, 'name')).toEqual({ key: 'name', dir: 1 });
    expect(nextSort({ key: 'name', dir: -1 }, 'email')).toEqual({ key: 'email', dir: 1 });
  });
});
