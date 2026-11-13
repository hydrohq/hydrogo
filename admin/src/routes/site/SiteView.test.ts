import { describe, expect, it } from 'vitest';
import {
  dirtyTranslationValues,
  mapTranslationRows,
  mergeSocialPlatforms,
  normalizeInternalHref,
  normalizeSocialUrl,
} from './siteHelpers';

describe('site settings helpers', () => {
  it('merges empty social profile list against supported platforms without dropping inputs', () => {
    const merged = mergeSocialPlatforms(['instagram', 'twitter_x', 'github'], []);
    expect(merged).toEqual([
      { platform: 'instagram', url: '' },
      { platform: 'twitter_x', url: '' },
      { platform: 'github', url: '' },
    ]);
  });

  it('merges saved social URLs onto platform order', () => {
    const merged = mergeSocialPlatforms(
      ['instagram', 'tiktok', 'twitter_x'],
      [{ platform: 'instagram', url: 'https://www.instagram.com/atelier' }],
    );
    expect(merged).toEqual([
      { platform: 'instagram', url: 'https://www.instagram.com/atelier' },
      { platform: 'tiktok', url: '' },
      { platform: 'twitter_x', url: '' },
    ]);
  });

  it('normalizes social URLs by upgrading http and stripping tracking params', () => {
    expect(normalizeSocialUrl(' http://www.instagram.com/atelier/?igsh=abc#top ')).toBe(
      'https://www.instagram.com/atelier/',
    );
    expect(normalizeSocialUrl('')).toBe('');
  });

  it('maps backend translation rows and extracts only dirty changes', () => {
    const rows = mapTranslationRows([
      { key: 'admin.dashboard', default: 'Dashboard', translation: 'Dashboard', missing: false },
      { key: 'admin.orders', default: 'Orders', translation: '', missing: true },
    ]);
    expect(rows[0].translation).toBe('Dashboard');
    expect(dirtyTranslationValues(rows)).toEqual({});

    rows[1].translation = 'Siparişler';
    expect(dirtyTranslationValues(rows)).toEqual({ 'admin.orders': 'Siparişler' });
  });

  it('normalizes relative navigation hrefs to start with slash', () => {
    expect(normalizeInternalHref('kategori/kemerler')).toBe('/kategori/kemerler');
    expect(normalizeInternalHref('/urunler')).toBe('/urunler');
    expect(normalizeInternalHref('https://example.com')).toBe('https://example.com');
  });
});
