/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';
import { en } from './en';
import { tr } from './tr';
import { interpolate, isLocale, pickUserChoice, resolveLocale } from './index.svelte';

describe('admin i18n', () => {
  it('has the same keys in every catalog', () => {
    expect(Object.keys(tr).sort()).toEqual(Object.keys(en).sort());
  });

  it('defines every key in exactly one area file', () => {
    const areas = import.meta.glob('./messages/*.ts', { eager: true }) as Record<string, { en: object }>;
    const seen = new Set<string>();
    for (const [file, mod] of Object.entries(areas)) {
      for (const key of Object.keys(mod.en)) {
        expect(seen.has(key), `${key} duplicated in ${file}`).toBe(false);
        seen.add(key);
      }
    }
    expect(seen.size).toBe(Object.keys(en).length);
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

  it('prefers the server-saved language over the browser copy', () => {
    expect(pickUserChoice('tr', 'en')).toBe('tr');
    expect(pickUserChoice(null, 'en')).toBe('en');
    expect(pickUserChoice(undefined, 'tr')).toBe('tr');
    expect(pickUserChoice('xx', 'tr')).toBe('tr');
    expect(pickUserChoice(null, null)).toBeNull();
    expect(resolveLocale(pickUserChoice(null, null), 'tr')).toBe('tr');
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
