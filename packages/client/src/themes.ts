export interface ThemeTokens {
  bg: string;
  surface: string;
  fg: string;
  muted: string;
  accent: string;
  font_display: "sans" | "serif" | "mono" | "display";
  font_body: "sans" | "serif" | "mono";
  radius: "none" | "sm" | "md" | "lg";
  header_variant: "split" | "centered" | "minimal";
}

export interface ThemeBlockBlueprint {
  type: string;
  label: string;
  props?: Record<string, unknown>;
}

export interface ThemeManifest {
  id: string;
  name: string;
  description: string;
  supported_presets: string[];
  required_modules?: string[];
  recommended_layout: string;
  default_blocks: ThemeBlockBlueprint[];
  default_tokens: ThemeTokens;
}

export interface SiteThemesResponse {
  themes: ThemeManifest[];
  active: ThemeManifest;
  tokens: ThemeTokens;
  preview: boolean;
}

const RADIUS_MAP: Record<ThemeTokens["radius"], string> = {
  none: "0px",
  sm: "0.25rem",
  md: "0.5rem",
  lg: "0.75rem",
};

const FONT_MAP: Record<ThemeTokens["font_display"], string> = {
  sans: "ui-sans-serif, system-ui, sans-serif",
  serif: "ui-serif, Georgia, Cambria, serif",
  mono: "ui-monospace, SFMono-Regular, Menlo, monospace",
  display: "ui-sans-serif, system-ui, sans-serif",
};

/**
 * Converts validated HydroGo ThemeTokens into CSS custom properties (`--hydro-*`)
 * for runtime storefront theming without rebuilding.
 */
export function themeTokensToCSSVars(tokens: ThemeTokens): Record<string, string> {
  return {
    "--hydro-bg": tokens.bg,
    "--hydro-surface": tokens.surface,
    "--hydro-fg": tokens.fg,
    "--hydro-muted": tokens.muted,
    "--hydro-accent": tokens.accent,
    "--hydro-font-display": FONT_MAP[tokens.font_display] ?? FONT_MAP.sans,
    "--hydro-font-body": FONT_MAP[tokens.font_body] ?? FONT_MAP.sans,
    "--hydro-radius": RADIUS_MAP[tokens.radius] ?? RADIUS_MAP.md,
  };
}

/**
 * Formats ThemeTokens as an inline `style` attribute string for SSR root elements.
 */
export function themeTokensToStyleString(tokens: ThemeTokens): string {
  return Object.entries(themeTokensToCSSVars(tokens))
    .map(([key, value]) => `${key}: ${value}`)
    .join("; ");
}
