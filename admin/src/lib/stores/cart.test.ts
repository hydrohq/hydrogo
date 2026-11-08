import { describe, it, expect, beforeEach } from 'vitest';
import { cart } from './cart.svelte';

describe('cart store', () => {
  beforeEach(() => {
    cart.clear();
    cart.isOpen = false;
  });

  it('initializes empty', () => {
    expect(cart.items).toEqual([]);
    expect(cart.totalCount).toBe(0);
    expect(cart.totalCents).toBe(0);
  });

  it('adds items and calculates total', () => {
    cart.addItem({
      id: 'prod-1',
      name: 'Test Product',
      priceCents: 15000,
      currency: 'TRY',
    });

    expect(cart.items.length).toBe(1);
    expect(cart.totalCount).toBe(1);
    expect(cart.totalCents).toBe(15000);
    expect(cart.isOpen).toBe(true);

    // Add again increments quantity
    cart.addItem({
      id: 'prod-1',
      name: 'Test Product',
      priceCents: 15000,
    });

    expect(cart.items.length).toBe(1);
    expect(cart.totalCount).toBe(2);
    expect(cart.totalCents).toBe(30000);
  });

  it('updates quantity and removes item when reaching zero', () => {
    cart.addItem({
      id: 'prod-1',
      name: 'Test Product',
      priceCents: 5000,
    });

    cart.updateQuantity('prod-1', 1);
    expect(cart.totalCount).toBe(2);

    cart.updateQuantity('prod-1', -1);
    expect(cart.totalCount).toBe(1);

    cart.updateQuantity('prod-1', -1);
    expect(cart.items.length).toBe(0);
    expect(cart.totalCount).toBe(0);
  });

  it('removes item directly', () => {
    cart.addItem({
      id: 'prod-1',
      name: 'Test 1',
      priceCents: 1000,
    });
    cart.addItem({
      id: 'prod-2',
      name: 'Test 2',
      priceCents: 2000,
    });

    expect(cart.items.length).toBe(2);
    cart.removeItem('prod-1');
    expect(cart.items.length).toBe(1);
    expect(cart.items[0].id).toBe('prod-2');
  });

  it('toggles open state', () => {
    expect(cart.isOpen).toBe(false);
    cart.toggle();
    expect(cart.isOpen).toBe(true);
    cart.toggle(false);
    expect(cart.isOpen).toBe(false);
  });
});
