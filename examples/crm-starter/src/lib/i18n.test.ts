import { describe, expect, it } from 'vitest';
import { dealCount, messages, pickLang } from './i18n';

const keys = (value: Record<string, unknown>, prefix = ''): string[] =>
  Object.entries(value).flatMap(([key, entry]) => (typeof entry === 'object' && entry ? keys(entry as Record<string, unknown>, `${prefix}${key}.`) : [`${prefix}${key}`]));

describe('i18n', () => {
  it('has the same keys in every language', () => {
    expect(keys(messages.tr).sort()).toEqual(keys(messages.en).sort());
  });
  it('falls back for unknown languages', () => {
    expect(pickLang('fr')).toBe('en');
    expect(pickLang('tr')).toBe('tr');
    expect(pickLang(undefined, 'tr')).toBe('tr');
  });
  it('counts deals with the right noun', () => {
    expect(dealCount(messages.en, 1)).toBe('1 deal');
    expect(dealCount(messages.en, 3)).toBe('3 deals');
    expect(dealCount(messages.tr, 3)).toBe('3 fırsat');
  });
});
