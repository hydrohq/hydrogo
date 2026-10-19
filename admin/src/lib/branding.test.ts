import { describe, expect, it } from 'vitest';
import { hexToHslTriple } from './branding';

describe('hexToHslTriple', () => {
  it('converts a colour and picks a readable foreground side', () => {
    expect(hexToHslTriple('#ff0000')).toEqual({ triple: '0 100% 50%', light: false });
    expect(hexToHslTriple('#ffffff')?.light).toBe(true);
  });
  it('rejects invalid input', () => {
    expect(hexToHslTriple('red')).toBeNull();
    expect(hexToHslTriple('#fff')).toBeNull();
  });
});
