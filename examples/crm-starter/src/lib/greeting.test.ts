import { describe, expect, it } from 'vitest';
import { firstName, greetingKey } from './greeting';

describe('greeting', () => {
  it('follows the hour', () => {
    expect(greetingKey(6)).toBe('greetMorning');
    expect(greetingKey(12)).toBe('greetAfternoon');
    expect(greetingKey(17)).toBe('greetAfternoon');
    expect(greetingKey(18)).toBe('greetEvening');
    expect(greetingKey(2)).toBe('greetMorning');
  });
  it('takes a first name from an address', () => {
    expect(firstName('jane.doe@example.com')).toBe('Jane');
    expect(firstName('admin@example.com')).toBe('Admin');
    expect(firstName('')).toBe('');
    expect(firstName(null)).toBe('');
  });
});
