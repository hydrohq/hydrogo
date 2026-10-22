import { describe, expect, it } from 'vitest';
import { messages, pickLang } from './i18n';

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
});
