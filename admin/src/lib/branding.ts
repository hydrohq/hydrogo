export interface Branding {
  name: string;
  logo_url: string;
  color: string;
}

/** Converts #rrggbb to the "h s% l%" triple the theme variables use. */
export function hexToHslTriple(hex: string): { triple: string; light: boolean } | null {
  const m = /^#([0-9a-f]{6})$/i.exec(hex);
  if (!m) return null;
  const n = parseInt(m[1], 16);
  const r = ((n >> 16) & 255) / 255;
  const g = ((n >> 8) & 255) / 255;
  const b = (n & 255) / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const l = (max + min) / 2;
  const d = max - min;
  let h = 0;
  let s = 0;
  if (d !== 0) {
    s = d / (1 - Math.abs(2 * l - 1));
    if (max === r) h = ((g - b) / d) % 6;
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    h = (h * 60 + 360) % 360;
  }
  return { triple: `${Math.round(h)} ${Math.round(s * 100)}% ${Math.round(l * 100)}%`, light: l > 0.6 };
}

/** Applies the accent colour to the theme, or restores the defaults. */
export function applyBrandColor(color: string | undefined, root: HTMLElement = document.documentElement) {
  const parsed = color ? hexToHslTriple(color) : null;
  if (!parsed) {
    root.style.removeProperty('--primary');
    root.style.removeProperty('--primary-foreground');
    return;
  }
  root.style.setProperty('--primary', parsed.triple);
  root.style.setProperty('--primary-foreground', parsed.light ? '240 6% 10%' : '0 0% 98%');
}
