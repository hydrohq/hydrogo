import { describe, expect, it } from 'vitest';
import { productTabs } from './productTabs';

describe('productTabs', () => {
  it('registers only well-formed tabs (core ships none; deployments add them under ./extensions)', () => {
    expect(Array.isArray(productTabs)).toBe(true);
    for (const tab of productTabs) {
      expect(tab.id).toBeTruthy();
      expect(tab.label).toBeTruthy();
      expect(tab.component).toBeTruthy();
    }
  });
});
