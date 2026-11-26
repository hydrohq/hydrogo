export interface VariantCombo {
  color: string;
  size: string;
}

const TURKISH_MAP: Record<string, string> = {
  ç: 'c',
  Ç: 'C',
  ğ: 'g',
  Ğ: 'G',
  ı: 'i',
  I: 'I',
  İ: 'I',
  i: 'i',
  ö: 'o',
  Ö: 'O',
  ş: 's',
  Ş: 'S',
  ü: 'u',
  Ü: 'U',
};

/**
 * Transliterates Turkish characters to ASCII counterparts:
 * ç->c, ğ->g, ı->i, ö->o, ş->s, ü->u (and uppercase variants).
 */
export function transliterateTurkish(text: string): string {
  if (!text) return '';
  return text
    .split('')
    .map((ch) => TURKISH_MAP[ch] ?? ch)
    .join('');
}

/**
 * Generates an uppercase SKU from slug, color, and size:
 * - Translates Turkish characters
 * - Uppercases everything
 * - Collapses non [A-Z0-9] characters into '-'
 * - Trims leading and trailing hyphens
 */
export function generateVariantSku(slug: string, color: string, size: string): string {
  const s = slug ? slug.trim() : 'PRD';
  const c = color ? color.trim() : '';
  const sz = size ? size.trim() : '';
  const raw = `${s}-${c}-${sz}`;

  return transliterateTurkish(raw)
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * Computes missing variant combinations against existing variants.
 * Supports 2D (color x size) as well as single-axis (size-only or color-only) products.
 */
export function getMissingCombos(
  colors: string[],
  sizes: string[],
  variants: Array<{ color?: string | null; size?: string | null }>
): VariantCombo[] {
  const cleanColors = colors.map((c) => c.trim()).filter(Boolean);
  const cleanSizes = sizes.map((s) => s.trim()).filter(Boolean);
  if (!cleanColors.length && !cleanSizes.length) return [];

  const existing = new Set<string>();
  for (const v of variants) {
    const c = (v.color ?? '').trim();
    const s = (v.size ?? '').trim();
    if (c || s) {
      existing.add(`${c}:${s}`);
    }
  }

  const missing: VariantCombo[] = [];
  if (cleanColors.length && cleanSizes.length) {
    for (const c of cleanColors) {
      for (const s of cleanSizes) {
        if (!existing.has(`${c}:${s}`)) {
          missing.push({ color: c, size: s });
        }
      }
    }
  } else if (cleanSizes.length) {
    for (const s of cleanSizes) {
      if (!existing.has(`:${s}`)) {
        missing.push({ color: '', size: s });
      }
    }
  } else {
    for (const c of cleanColors) {
      if (!existing.has(`${c}:`)) {
        missing.push({ color: c, size: '' });
      }
    }
  }

  return missing;
}
