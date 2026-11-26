import { describe, expect, it } from 'vitest';
import {
  generateVariantSku,
  getMissingCombos,
  transliterateTurkish,
} from './variantSku';

describe('variantSku helpers', () => {
  it('transliterates Turkish characters to ASCII', () => {
    expect(transliterateTurkish('çğıöşüÇĞIİÖŞÜ')).toBe('cgiosuCGIIOSU');
    expect(transliterateTurkish('ılık çay')).toBe('ilik cay');
  });

  it('generates uppercase SKU transliterating Turkish and collapsing non-alphanumeric chars', () => {
    expect(generateVariantSku('deri-kemer', 'Kahverengi', '105')).toBe('DERI-KEMER-KAHVERENGI-105');
    expect(generateVariantSku('kemer', 'Açık Sarı', 'Ölçü 2')).toBe('KEMER-ACIK-SARI-OLCU-2');
    expect(generateVariantSku('silk-shirt', 'Ilık Mavi', 'Şık S')).toBe('SILK-SHIRT-ILIK-MAVI-SIK-S');
    expect(generateVariantSku('', 'Kırmızı', 'M')).toBe('PRD-KIRMIZI-M');
  });

  it('computes missing color x size combos', () => {
    const colors = ['Siyah', 'Kırmızı'];
    const sizes = ['S', 'M'];
    const existing = [
      { color: 'Siyah', size: 'S' },
      { color: 'Kırmızı', size: 'M' },
    ];

    const missing = getMissingCombos(colors, sizes, existing);
    expect(missing).toEqual([
      { color: 'Siyah', size: 'M' },
      { color: 'Kırmızı', size: 'S' },
    ]);
  });

  it('returns empty missing combos when all combos exist', () => {
    const colors = ['Siyah'];
    const sizes = ['S'];
    const existing = [{ color: 'Siyah', size: 'S' }];

    expect(getMissingCombos(colors, sizes, existing)).toEqual([]);
  });

  it('supports single-axis (size-only or color-only) combos and returns empty when both are empty', () => {
    expect(getMissingCombos([], [], [])).toEqual([]);
    expect(getMissingCombos([], ['100 gr', '250 gr'], [{ size: '100 gr' }])).toEqual([
      { color: '', size: '250 gr' },
    ]);
    expect(getMissingCombos(['Siyah', 'Beyaz'], [], [{ color: 'Siyah' }])).toEqual([
      { color: 'Beyaz', size: '' },
    ]);
  });
});
