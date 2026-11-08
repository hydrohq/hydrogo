// @vitest-environment jsdom
import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import StorefrontApp from './StorefrontApp.svelte';
import { cart } from '$lib/stores/cart.svelte';

describe('StorefrontApp component', () => {
  beforeEach(() => {
    cart.clear();
    cart.isOpen = false;
    window.__HYDRO_DATA__ = {
      store: {
        name: 'Mio Silk Boutique',
        tagline: 'Lüks ipek eşarplar ve butik koleksiyonlar.',
        phone: '05551234567',
        currency: 'TRY',
      },
      products: [
        {
          id: 'p-1',
          name: 'Saf İpek Şal',
          price: 1250,
          price_cents: 125000,
          currency: 'TRY',
          category: 'İpek Şal',
          description: 'El dokuması saf ipek şal.',
          in_stock: true,
        },
        {
          id: 'p-2',
          name: 'Kaşmir Şal',
          price: 2450,
          price_cents: 245000,
          currency: 'TRY',
          category: 'Kaşmir',
          description: 'Hafif ve sıcak tutan kaşmir.',
          in_stock: true,
        },
      ],
    };
  });

  it('renders store name, tagline and products from SSR data', () => {
    render(StorefrontApp);
    expect(screen.getAllByText('Mio Silk Boutique').length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Lüks ipek eşarplar/).length).toBeGreaterThan(0);
    expect(screen.getByText('Saf İpek Şal')).toBeTruthy();
    expect(screen.getByText('Kaşmir Şal')).toBeTruthy();
  });

  it('filters products by category tab', async () => {
    render(StorefrontApp);
    const kashmirTab = screen.getByRole('button', { name: 'Kaşmir' });
    await fireEvent.click(kashmirTab);

    expect(screen.queryByText('Saf İpek Şal')).toBeNull();
    expect(screen.getByText('Kaşmir Şal')).toBeTruthy();
  });

  it('adds item to cart and opens drawer', async () => {
    render(StorefrontApp);
    const addButtons = screen.getAllByLabelText('Sepete Ekle');
    await fireEvent.click(addButtons[0]);

    expect(cart.totalCount).toBe(1);
    expect(cart.isOpen).toBe(true);
    expect(screen.getByRole('dialog', { name: 'Alışveriş Sepeti' })).toBeTruthy();
  });
});
