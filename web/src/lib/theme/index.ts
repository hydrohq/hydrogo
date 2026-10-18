/**
 * Centralized Design System & Theme Configuration for HydroPost
 * All colors, buttons, cards, typography, and layout presets are defined here
 * as the single source of truth for the entire application.
 */

export const THEME_CONFIG = {
  portalName: 'HYDROPOST',
  portalTagline: 'BAĞIMSIZ HABER PORTALI',
  slogan: 'Hızlı, Güvenilir, Tarafsız Haber',

  // Core Brand Colors
  colors: {
    primary: '#dc2626', // Brand Red
    primaryDark: '#b91c1c', // Darker Red for hover/active
    primaryLight: '#fef2f2', // Tint for subtle backgrounds
    primaryBorder: '#fecaca', // Border for primary tags

    accentYellow: '#ffe600', // Manşet Headline Accent Yellow
    accentYellowDark: '#eab308',

    background: '#ffffff', // Pure clean white
    surface: '#fafafa', // Subtle section backgrounds (zinc-50)
    surfaceMuted: '#f4f4f5', // Neutral containers (zinc-100)

    border: '#e4e4e7', // Standard card & divider border (zinc-200)
    borderSubtle: '#f4f4f5', // Light inner divider
    borderStrong: '#d4d4d8', // Input & active borders (zinc-300)

    textHeading: '#09090b', // Deep rich black for titles (zinc-950)
    textBody: '#27272a', // Readable dark grey for copy (zinc-800)
    textMuted: '#71717a', // Metadata & caption grey (zinc-500)
    textSubtle: '#a1a1aa', // Timestamps & minor labels (zinc-400)
  },

  // Button Presets
  button: {
    primary:
      'bg-red-600 hover:bg-red-700 active:bg-red-800 text-white font-bold text-sm px-4 py-2 rounded-xl shadow-xs transition-colors cursor-pointer inline-flex items-center justify-center gap-2 select-none',
    secondary:
      'bg-zinc-100 hover:bg-zinc-200 active:bg-zinc-300 text-zinc-900 font-semibold text-sm px-4 py-2 rounded-xl border border-zinc-200 transition-colors cursor-pointer inline-flex items-center justify-center gap-2 select-none',
    outline:
      'bg-white hover:bg-zinc-50 active:bg-zinc-100 text-zinc-800 font-semibold text-sm px-4 py-2 rounded-xl border border-zinc-300 hover:border-zinc-400 transition-colors cursor-pointer inline-flex items-center justify-center gap-2 select-none',
    ghost:
      'text-zinc-600 hover:text-zinc-950 hover:bg-zinc-100 active:bg-zinc-200 font-semibold text-sm px-3 py-1.5 rounded-lg transition-colors cursor-pointer inline-flex items-center justify-center gap-1.5 select-none',
    danger:
      'bg-rose-600 hover:bg-rose-700 active:bg-rose-800 text-white font-bold text-sm px-4 py-2 rounded-xl shadow-xs transition-colors cursor-pointer inline-flex items-center justify-center gap-2 select-none',
    pill:
      'rounded-full px-4 py-1.5 text-xs font-bold transition-colors cursor-pointer inline-flex items-center justify-center gap-1.5 select-none',
    sm: 'text-xs px-2.5 py-1 rounded-lg',
    md: 'text-sm px-4 py-2 rounded-xl',
    lg: 'text-base px-5 py-2.5 rounded-xl font-bold',
  },

  // Card Presets
  card: {
    base:
      'bg-white border border-zinc-200 rounded-2xl shadow-xs overflow-hidden transition-colors',
    interactive:
      'group bg-white border border-zinc-200 rounded-2xl shadow-xs hover:border-zinc-300 transition-colors overflow-hidden',
    subtle:
      'bg-zinc-50/75 border border-zinc-200/80 rounded-xl p-4',
    stickySidebar:
      'bg-white border border-zinc-200 rounded-2xl shadow-xs p-5 sticky top-20',
  },

  // Badge Presets
  badge: {
    primary:
      'inline-flex items-center px-2 py-0.5 rounded text-[11px] font-bold tracking-wider uppercase bg-red-50 text-red-700 border border-red-200',
    live:
      'inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200',
    breaking:
      'inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-black uppercase tracking-widest bg-red-600 text-white',
    categoryDefault:
      'inline-flex items-center px-2 py-0.5 rounded text-[11px] font-bold tracking-wider uppercase bg-zinc-100 text-zinc-700 border border-zinc-200',
  },

  // Headline & Text Styling
  headline: {
    cardTitle:
      'text-base font-bold text-zinc-900 group-hover:underline decoration-zinc-900 underline-offset-2 transition-none line-clamp-2 leading-snug',
    heroTitleWhite:
      'text-xl sm:text-2xl md:text-4xl font-extrabold text-white leading-tight drop-shadow-md',
    heroPunchlineYellow:
      'text-xl sm:text-2xl md:text-4xl font-black text-[#ffe600] uppercase tracking-wide leading-tight drop-shadow-lg',
    metaText:
      'text-xs text-zinc-500 flex items-center gap-2',
  },
} as const;

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
