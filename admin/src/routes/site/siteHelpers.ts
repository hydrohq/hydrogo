export interface SocialProfile {
  platform: string;
  url: string;
}

export interface BusinessProfileForm {
  name: string;
  legal_name: string;
  tax_id: string;
  tax_office: string;
  mersis_no: string;
  kep_address: string;
  email: string;
  phone: string;
  website: string;
  address: string;
  city: string;
  country: string;
  postal_code: string;
  timezone: string;
}

export interface BackendTranslationRow {
  key: string;
  default?: string;
  translation?: string;
  missing?: boolean;
}

export interface EditableTranslationRow {
  key: string;
  default: string;
  translation: string;
  initial: string;
  missing: boolean;
}

export const DEFAULT_SOCIAL_PLATFORMS = [
  'instagram',
  'tiktok',
  'facebook',
  'twitter_x',
  'youtube',
  'linkedin',
  'github',
] as const;

export const SOCIAL_PLATFORM_LABELS: Record<string, string> = {
  instagram: 'Instagram',
  tiktok: 'TikTok',
  facebook: 'Facebook',
  twitter_x: 'X (Twitter)',
  youtube: 'YouTube',
  linkedin: 'LinkedIn',
  github: 'GitHub',
};

export function socialPlatformPlaceholder(platform: string): string {
  switch (platform) {
    case 'twitter_x':
      return 'https://x.com/markaniz';
    case 'instagram':
      return 'https://www.instagram.com/markaniz';
    case 'tiktok':
      return 'https://www.tiktok.com/@markaniz';
    case 'youtube':
      return 'https://www.youtube.com/@markaniz';
    case 'facebook':
      return 'https://www.facebook.com/markaniz';
    case 'linkedin':
      return 'https://www.linkedin.com/company/markaniz';
    case 'github':
      return 'https://github.com/markaniz';
    default:
      return `https://${platform}.com/...`;
  }
}

export function mergeSocialPlatforms(
  platforms?: string[] | null,
  profiles?: SocialProfile[] | null,
): SocialProfile[] {
  const list = platforms && platforms.length > 0 ? platforms : [...DEFAULT_SOCIAL_PLATFORMS];
  const byPlatform = new Map<string, string>();
  for (const item of profiles ?? []) {
    if (item?.platform) byPlatform.set(item.platform, item.url ?? '');
  }
  return list.map((platform) => ({
    platform,
    url: byPlatform.get(platform) ?? '',
  }));
}

export function normalizeSocialUrl(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return '';
  const candidate = trimmed.startsWith('http://') ? `https://${trimmed.slice(7)}` : trimmed;
  try {
    const parsed = new URL(candidate);
    parsed.search = '';
    parsed.hash = '';
    return parsed.toString();
  } catch {
    return trimmed;
  }
}

export function mapTranslationRows(rows?: BackendTranslationRow[] | null): EditableTranslationRow[] {
  return (rows ?? []).map((row) => ({
    key: row.key,
    default: row.default ?? '',
    translation: row.translation ?? '',
    initial: row.translation ?? '',
    missing: Boolean(row.missing),
  }));
}

export function dirtyTranslationValues(items: EditableTranslationRow[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const item of items) {
    if (item.translation !== item.initial) {
      out[item.key] = item.translation;
    }
  }
  return out;
}

export function normalizeInternalHref(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed || trimmed.startsWith('/') || trimmed.startsWith('#') || trimmed.includes(':')) {
    return trimmed;
  }
  return `/${trimmed}`;
}
