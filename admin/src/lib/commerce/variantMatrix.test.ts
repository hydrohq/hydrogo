import { describe, expect, it } from 'vitest';
import { buildMatrix, sortSizes } from './variantMatrix';

describe('variantMatrix', () => {
  describe('sortSizes', () => {
    it('sorts numeric sizes numerically', () => {
      const input = ['42', '36', '40', '38'];
      expect(sortSizes(input)).toEqual(['36', '38', '40', '42']);
    });

    it('sorts non-numeric sizes alphabetically', () => {
      const input = ['XL', 'S', 'M', 'L', 'XS'];
      expect(sortSizes(input)).toEqual(['L', 'M', 'S', 'XL', 'XS']);
    });

    it('sorts numbers first numerically then non-numbers alphabetically in mixed input', () => {
      const input = ['XL', '38', 'S', '36'];
      expect(sortSizes(input)).toEqual(['36', '38', 'S', 'XL']);
    });
  });

  describe('buildMatrix', () => {
    it('returns empty matrix for empty array', () => {
      const result = buildMatrix([]);
      expect(result.colors).toEqual([]);
      expect(result.sizes).toEqual([]);
      expect(result.cells['anything']).toBeUndefined();
    });

    it('builds colors and sizes and enables lookup by color+size', () => {
      const variants = [
        { id: 'v1', product: 'p1', sku: 'SKU-RED-M', color: 'Red', size: 'M' },
        { id: 'v2', product: 'p1', sku: 'SKU-RED-S', color: 'Red', size: 'S' },
        { id: 'v3', product: 'p1', sku: 'SKU-BLU-M', color: 'Blue', size: 'M' },
      ];

      const matrix = buildMatrix(variants);

      expect(matrix.colors).toEqual(['Blue', 'Red']);
      expect(matrix.sizes).toEqual(['M', 'S']);

      // Lookups
      expect(matrix.cells['Red:M']).toEqual(variants[0]);
      expect(matrix.cells['Red:S']).toEqual(variants[1]);
      expect(matrix.cells['Blue:M']).toEqual(variants[2]);
      expect(matrix.cells['Blue:S']).toBeUndefined();

      // Nested or get helper lookup
      expect(matrix.cells.Blue?.M).toEqual(variants[2]);
      expect(matrix.cells.get?.('Red', 'M')).toEqual(variants[0]);
      expect(matrix.cells.get?.('Blue', 'S')).toBeUndefined();
    });

    it('handles numeric sizes in matrix', () => {
      const variants = [
        { id: 'v1', color: 'Black', size: '42' },
        { id: 'v2', color: 'Black', size: '38' },
        { id: 'v3', color: 'Black', size: '40' },
      ];

      const matrix = buildMatrix(variants);
      expect(matrix.sizes).toEqual(['38', '40', '42']);
      expect(matrix.cells['Black:38']).toEqual(variants[1]);
    });
  });
});
