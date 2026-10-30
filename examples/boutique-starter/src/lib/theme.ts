export interface StoreSettings {
  store_name: string;
  slogan?: string;
  logo?: string;
  whatsapp_number: string;
  instagram_handle?: string;
  announcement_bar?: string;
  free_shipping_threshold?: number;
  theme_preset?: 'minimal_cream' | 'noir_gold' | 'pastel_blush' | 'emerald_silk' | string;
  bg_primary_color?: string;
  text_primary_color?: string;
  accent_color?: string;
  font_heading?: string;
  border_radius_style?: 'sharp' | 'soft' | 'pill' | string;
  hero_banner_image?: string;
  hero_title?: string;
  hero_cta_text?: string;
}

export interface ThemeTokens {
  bgPrimary: string;
  textPrimary: string;
  accent: string;
  fontHeading: string;
  radius: string;
}

const PRESETS: Record<string, ThemeTokens> = {
  minimal_cream: {
    bgPrimary: '#FAF7F2',
    textPrimary: '#1C1917',
    accent: '#8C6D46',
    fontHeading: "'Cormorant Garamond', Georgia, serif",
    radius: '10px'
  },
  noir_gold: {
    bgPrimary: '#111111',
    textPrimary: '#F5F2EB',
    accent: '#D4AF37',
    fontHeading: "'Bodoni Moda', Georgia, serif",
    radius: '2px'
  },
  pastel_blush: {
    bgPrimary: '#FFF9F9',
    textPrimary: '#2D2326',
    accent: '#C97A8E',
    fontHeading: "'Playfair Display', Georgia, serif",
    radius: '18px'
  },
  emerald_silk: {
    bgPrimary: '#F4F8F6',
    textPrimary: '#102820',
    accent: '#1B5E43',
    fontHeading: "'Cormorant Garamond', Georgia, serif",
    radius: '8px'
  }
};

export function resolveThemeTokens(settings: StoreSettings): ThemeTokens {
  const base = PRESETS[settings.theme_preset ?? 'minimal_cream'] ?? PRESETS.minimal_cream;
  let radius = base.radius;
  if (settings.border_radius_style === 'sharp') radius = '2px';
  if (settings.border_radius_style === 'soft') radius = '10px';
  if (settings.border_radius_style === 'pill') radius = '20px';

  return {
    bgPrimary: settings.bg_primary_color || base.bgPrimary,
    textPrimary: settings.text_primary_color || base.textPrimary,
    accent: settings.accent_color || base.accent,
    fontHeading: settings.font_heading ? `'${settings.font_heading}', Georgia, serif` : base.fontHeading,
    radius
  };
}
