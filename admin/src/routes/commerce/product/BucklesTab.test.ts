// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import { i18n } from '$lib/i18n/index.svelte';
import BucklesTab from './BucklesTab.svelte';

vi.mock('$lib/api/client', () => {
  class MockApiClientError extends Error {
    code = 'ERROR';
    requestId = 'req-123';
    status = 500;
    constructor(error: any, status = 500) {
      super(error?.message || 'API request failed');
      this.name = 'ApiClientError';
      this.status = status;
    }
  }

  return {
    api: {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
    },
    ApiClientError: MockApiClientError,
  };
});

describe('BucklesTab', () => {
  const mockCategories = [
    { id: 'cat-belts', name: 'Kemerler', slug: 'kemerler' },
    { id: 'cat-tokalar', name: 'Tokalar', slug: 'tokalar' },
  ];

  const mockProducts = [
    {
      id: 'p1',
      name: 'Klasik Deri Kemer',
      category: 'cat-belts',
      base_price: 600,
      currency: 'TRY',
      status: 'published',
    },
    {
      id: 'b1',
      name: 'Gümüş Klasik Toka',
      category: 'cat-tokalar',
      base_price: 150,
      currency: 'TRY',
      status: 'published',
      cover_image: 'media-b1',
    },
    {
      id: 'b2',
      name: 'Antik Pirinç Toka',
      category: 'cat-tokalar',
      base_price: 180,
      currency: 'TRY',
      status: 'draft',
      cover_image: 'media-b2',
    },
    {
      id: 'b3',
      name: 'Mat Siyah Toka',
      category: 'cat-tokalar',
      base_price: 160,
      currency: 'TRY',
      status: 'published',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    i18n.locale = 'tr';

    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/categories')) {
        return Promise.resolve({ data: mockCategories });
      }
      if (url.includes('/api/v1/admin/content/products')) {
        return Promise.resolve({ data: mockProducts });
      }
      return Promise.resolve({ data: [] });
    });
  });

  afterEach(() => {
    i18n.locale = 'en';
  });

  it('renders buckle cards with preselected ones', async () => {
    const product = {
      id: 'p1',
      version: 2,
      name: 'Klasik Deri Kemer',
      category: 'cat-belts',
      base_price: 600,
      compatible_buckles: ['b1'],
    };

    render(BucklesTab, { props: { product } });

    await waitFor(() => {
      expect(screen.getByText('Gümüş Klasik Toka')).toBeTruthy();
    });
    expect(screen.getByText('Antik Pirinç Toka')).toBeTruthy();
    expect(screen.getByText('Mat Siyah Toka')).toBeTruthy();

    const btnB1 = screen.getByRole('button', { name: /Gümüş Klasik Toka/i });
    expect(btnB1.getAttribute('aria-pressed')).toBe('true');

    const btnB2 = screen.getByRole('button', { name: /Antik Pirinç Toka/i });
    expect(btnB2.getAttribute('aria-pressed')).toBe('false');

    expect(screen.getByText('1 seçildi')).toBeTruthy();

    const saveBtn = screen.getByRole('button', { name: 'Kaydet' });
    expect((saveBtn as HTMLButtonElement).disabled).toBe(true);
  });

  it('toggling enables save', async () => {
    const product = {
      id: 'p1',
      version: 2,
      name: 'Klasik Deri Kemer',
      category: 'cat-belts',
      base_price: 600,
      compatible_buckles: ['b1'],
    };

    render(BucklesTab, { props: { product } });

    const btnB2 = await screen.findByRole('button', { name: /Antik Pirinç Toka/i });
    const saveBtn = screen.getByRole('button', { name: 'Kaydet' });
    expect((saveBtn as HTMLButtonElement).disabled).toBe(true);

    await fireEvent.click(btnB2);
    expect(btnB2.getAttribute('aria-pressed')).toBe('true');
    expect(screen.getByText('2 seçildi')).toBeTruthy();
    expect((saveBtn as HTMLButtonElement).disabled).toBe(false);

    await fireEvent.click(btnB2);
    expect(btnB2.getAttribute('aria-pressed')).toBe('false');
    expect(screen.getByText('1 seçildi')).toBeTruthy();
    expect((saveBtn as HTMLButtonElement).disabled).toBe(true);
  });

  it('save PUTs expected body with version and calls onsaved', async () => {
    const onsaved = vi.fn();
    const updatedRecord = {
      id: 'p1',
      version: 3,
      name: 'Klasik Deri Kemer',
      category: 'cat-belts',
      base_price: 600,
      compatible_buckles: ['b1', 'b2'],
    };
    (api.put as any).mockResolvedValueOnce({ data: updatedRecord });

    const product = {
      id: 'p1',
      version: 2,
      name: 'Klasik Deri Kemer',
      category: 'cat-belts',
      base_price: 600,
      status: 'published',
      created_at: '2026-01-01',
      updated_at: '2026-01-02',
      compatible_buckles: ['b1'],
    };

    render(BucklesTab, { props: { product, onsaved } });

    const btnB2 = await screen.findByRole('button', { name: /Antik Pirinç Toka/i });
    await fireEvent.click(btnB2);

    const saveBtn = screen.getByRole('button', { name: 'Kaydet' });
    await fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(api.put).toHaveBeenCalledWith(
        '/api/v1/admin/content/products/p1',
        expect.objectContaining({
          name: 'Klasik Deri Kemer',
          category: 'cat-belts',
          base_price: 600,
          version: 2,
          compatible_buckles: ['b1', 'b2'],
        })
      );
    });

    const putPayload = (api.put as any).mock.calls[0][1];
    expect(putPayload.id).toBeUndefined();
    expect(putPayload.created_at).toBeUndefined();
    expect(putPayload.updated_at).toBeUndefined();
    expect(putPayload.status).toBeUndefined();

    expect(onsaved).toHaveBeenCalledWith(updatedRecord);
    expect(screen.getByText('Uyumlu tokalar başarıyla kaydedildi.')).toBeTruthy();
  });

  it('buckle product itself shows the note', async () => {
    const buckleProduct = {
      id: 'b1',
      version: 1,
      name: 'Gümüş Klasik Toka',
      category: 'cat-tokalar',
    };

    render(BucklesTab, { props: { product: buckleProduct } });

    await waitFor(() => {
      expect(screen.getByTestId('buckle-product-note')).toBeTruthy();
    });
    expect(
      screen.getByText(/bu ürün toka kategorisinde yer aldığı için uyumlu toka seçilemez/i)
    ).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Kaydet' })).toBeNull();
  });

  it('buckle product with category object shows the note', async () => {
    const buckleProduct = {
      id: 'b1',
      version: 1,
      name: 'Gümüş Klasik Toka',
      category: { id: 'cat-tokalar', slug: 'tokalar' },
    };

    render(BucklesTab, { props: { product: buckleProduct } });

    await waitFor(() => {
      expect(screen.getByTestId('buckle-product-note')).toBeTruthy();
    });
    expect(
      screen.getByText(/bu ürün toka kategorisinde yer aldığı için uyumlu toka seçilemez/i)
    ).toBeTruthy();
  });

  it('falls back to all other products with hint if tokalar category is absent', async () => {
    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/categories')) {
        return Promise.resolve({
          data: [{ id: 'cat-other', name: 'Diğer', slug: 'diger' }],
        });
      }
      if (url.includes('/api/v1/admin/content/products')) {
        return Promise.resolve({
          data: [
            { id: 'p1', name: 'Current Belt', category: 'cat-other' },
            { id: 'p2', name: 'Other Item 1', category: 'cat-other', base_price: 100 },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });

    const product = {
      id: 'p1',
      version: 1,
      name: 'Current Belt',
      category: 'cat-other',
    };

    render(BucklesTab, { props: { product } });

    await waitFor(() => {
      expect(screen.getByText('Other Item 1')).toBeTruthy();
    });
    expect(screen.getByText(/"tokalar" kategorisi bulunamadı/i)).toBeTruthy();
  });
});
