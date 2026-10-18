/**
 * Centralized Design System & Theme Configuration for HydroPost
 * All colors, buttons, cards, typography, and layout presets are defined here
 * as the single source of truth for the entire application.
 */

export const THEME_CONFIG = {
  portalName: 'HYDROPOST',
  portalTagline: 'BAĞIMSIZ HABER PORTALI',
  slogan: 'Hızlı, Güvenilir, Tarafsız Haber',

  // Core Brand Colors — warm paper ground, single ink-red accent, muted gold
  // for the manşet punchline (desaturated from the original neon #ffe600).
  colors: {
    primary: '#b8231f', // Ink red
    primaryDark: '#8f1a17',
    primaryLight: '#fbeceb',
    primaryBorder: '#ecc2c0',

    accentGold: '#d4a62c', // Manşet punchline — muted gold, not neon yellow
    accentGoldDeep: '#a9810f',

    background: '#faf8f4', // Warm paper, not clinical white
    surface: '#ffffff',
    surfaceMuted: '#f2ede4',

    border: '#e6ddcd',
    borderSubtle: '#efe9dc',
    borderStrong: '#d6c9af',

    textHeading: '#1c1917',
    textBody: '#3a332c',
    textMuted: '#76695a',
    textSubtle: '#a79a89',
  },

  // Category badge palette — intentionally limited to 5 entries so section
  // tags read as one deliberate system instead of a random Tailwind hue per
  // category. Breaking/gündem always uses the primary brand red; everything
  // else maps to one of these four accents in `getCategoryBadge` (api.ts).
  categoryPalette: {
    gundem: 'red', // primary brand red — breaking, siyaset, haber
    ekonomiSpor: 'amber-700', // ekonomi, ticaret, finans, spor
    teknoloji: 'slate-700', // teknoloji, ürün, tasarım, altyapı
    dunyaSaglik: 'teal-700', // dünya, sağlık, eczane, rehberler
    guvenlik: 'zinc-800', // güvenlik, mimari
  },

} as const;

// The actual single source of truth for component styling is the CSS custom
// property set in src/routes/layout.css (--hp-primary, --hp-bg, --hp-ink,
// --hp-font-display, ...). Components read it via Tailwind arbitrary values
// (`bg-[var(--hp-surface)]`) instead of duplicating a parallel TS preset
// table here — a string table nobody imports drifts from what's on screen,
// which is exactly how the old palette went inconsistent.

export type ThemeConfig = typeof THEME_CONFIG;

/**
 * Splits or formats news titles for Turkish portal hero sliders.
 * Produces:
 * - lead: White bold upper text
 * - punchline: Yellow extrabold punchline in uppercase
 */
export function splitMansetTitle(
  title: string = '',
  category: string = ''
): { lead: string; punchline: string } {
  const clean = (title || '').trim();
  if (!clean) {
    return {
      lead: category ? `${category.toLocaleUpperCase('tr-TR')} HABERİ` : 'SON DAKİKA GELİŞMESİ',
      punchline: 'DETAYLAR İÇİN TIKLAYIN',
    };
  }

  // 1. Colon check e.g. "ABD'de enerji devi: Hakan Özmen'e kritik görev"
  if (clean.includes(':')) {
    const parts = clean.split(':');
    return {
      lead: parts[0].trim(),
      punchline: parts.slice(1).join(':').trim().toLocaleUpperCase('tr-TR'),
    };
  }

  // 2. Dash check e.g. "Merkez Bankası Faizi Sabit Bıraktı - İşte Detaylar"
  if (clean.includes(' - ') || clean.includes(' — ')) {
    const sep = clean.includes(' — ') ? ' — ' : ' - ';
    const parts = clean.split(sep);
    return {
      lead: parts[0].trim(),
      punchline: parts.slice(1).join(sep).trim().toLocaleUpperCase('tr-TR'),
    };
  }

  // 3. Question or exclamation mark split
  if (clean.includes('?') && !clean.endsWith('?')) {
    const parts = clean.split('?');
    return {
      lead: `${parts[0].trim()}?`,
      punchline: parts.slice(1).join('?').trim().toLocaleUpperCase('tr-TR'),
    };
  }

  // 4. Natural word split if 4+ words
  const words = clean.split(/\s+/);
  if (words.length >= 4) {
    const splitIndex = Math.ceil(words.length / 2);
    return {
      lead: words.slice(0, splitIndex).join(' '),
      punchline: words.slice(splitIndex).join(' ').toLocaleUpperCase('tr-TR'),
    };
  }

  // 5. Short title fallback
  return {
    lead: category ? `${category.toLocaleUpperCase('tr-TR')} GÜNDEMİ` : 'GÜNÜN ÖNEMLİ GELİŞMESİ',
    punchline: clean.toLocaleUpperCase('tr-TR'),
  };
}
