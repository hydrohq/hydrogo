export interface MatrixVariant {
  id: string;
  product?: string | { id: string } | null;
  sku?: string | null;
  size?: string | null;
  color?: string | null;
  [key: string]: unknown;
}

export interface MatrixResult<T extends MatrixVariant = MatrixVariant> {
  colors: string[];
  sizes: string[];
  cells: Record<string, T | undefined> & {
    get?: (color: string, size: string) => T | undefined;
    [key: string]: any;
  };
}

/**
 * Sorts size strings. When sizes are numbers, sorts numerically; otherwise alphabetically.
 */
export function sortSizes(sizes: string[]): string[] {
  const isNum = (val: string) => val.trim() !== '' && !Number.isNaN(Number(val));
  return [...sizes].sort((a, b) => {
    const aIsNum = isNum(a);
    const bIsNum = isNum(b);
    if (aIsNum && bIsNum) {
      return Number(a) - Number(b);
    }
    if (aIsNum) return -1;
    if (bIsNum) return 1;
    return a.localeCompare(b);
  });
}

/**
 * Builds a 2D matrix structure of colors x sizes from an array of variants.
 */
export function buildMatrix<T extends MatrixVariant = MatrixVariant>(variants: T[]): MatrixResult<T> {
  const colorSet = new Set<string>();
  const sizeSet = new Set<string>();
  const cellMap: Record<string, T> = {};
  const nested: Record<string, Record<string, T>> = {};

  for (const v of variants) {
    const color = v.color !== undefined && v.color !== null ? String(v.color).trim() : '';
    const size = v.size !== undefined && v.size !== null ? String(v.size).trim() : '';

    colorSet.add(color);
    sizeSet.add(size);

    const keys = [
      `${color}:${size}`,
      `${color}|${size}`,
      `${color}:::${size}`,
      `${color}+${size}`,
    ];
    for (const k of keys) {
      cellMap[k] = v;
    }

    if (!nested[color]) {
      nested[color] = {};
    }
    nested[color][size] = v;
  }

  const colors = Array.from(colorSet).sort((a, b) => a.localeCompare(b));
  const sizes = sortSizes(Array.from(sizeSet));

  const handler: ProxyHandler<Record<string, any>> = {
    get(target, prop, receiver) {
      if (typeof prop === 'string') {
        if (prop === 'get') {
          return (c: string, s: string) =>
            cellMap[`${c}:${s}`] ?? cellMap[`${c}:::${s}`] ?? nested[c]?.[s];
        }
        if (prop in cellMap) {
          return cellMap[prop];
        }
        if (prop in nested) {
          return nested[prop];
        }
      }
      return Reflect.get(target, prop, receiver);
    },
  };

  const proxyCells = new Proxy(cellMap, handler);

  return {
    colors,
    sizes,
    cells: proxyCells,
  };
}
