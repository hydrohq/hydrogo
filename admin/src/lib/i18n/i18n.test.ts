import { describe, expect, it } from 'vitest';
import { en } from './en';
import { tr } from './tr';
import { interpolate, isLocale, resolveLocale } from './index.svelte';

describe('admin i18n', () => {
  it('has the same keys in every catalog', () => {
    expect(Object.keys(tr).sort()).toEqual(Object.keys(en).sort());
  });

  it('keeps placeholders identical across languages', () => {
    const placeholders = (text: string) => (text.match(/\{\w+\}/g) ?? []).sort();
    for (const key of Object.keys(en) as (keyof typeof en)[]) {
      expect(placeholders(tr[key]), key).toEqual(placeholders(en[key]));
    }
  });

  it('prefers the user choice, then the project default', () => {
    expect(resolveLocale('tr', 'en')).toBe('tr');
    expect(resolveLocale(null, 'tr')).toBe('tr');
    expect(resolveLocale('xx', 'tr')).toBe('tr');
    expect(resolveLocale(null, 'xx')).toBe('en');
  });

  it('interpolates named parameters and leaves unknown ones', () => {
    expect(interpolate('Active ({count})', { count: 3 })).toBe('Active (3)');
    expect(interpolate('Hi {name}', {})).toBe('Hi {name}');
  });

  it('validates locale codes', () => {
    expect(isLocale('en')).toBe(true);
    expect(isLocale('de')).toBe(false);
  });
});
