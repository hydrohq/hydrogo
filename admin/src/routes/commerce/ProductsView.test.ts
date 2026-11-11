// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { t } from '$lib/i18n/index.svelte';
import { api } from '$lib/api/client';
import { auth } from '$lib/stores/auth.svelte';
import { router } from '$lib/router.svelte';
import ProductsView from './ProductsView.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn().mockResolvedValue({}),
  },
  ApiClientError: class ApiClientError extends Error {
    requestId = '';
  },
}));

describe('ProductsView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    auth.user = { role: 'admin' } as any;
    auth.permissions = ['admin.commerce.write', 'admin.content.write'];
    vi.spyOn(router, 'navigate').mockImplementation(() => {});

    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/categories')) {
        return Promise.resolve({
          data: [
            { id: 'cat1', name: 'Clothing', slug: 'clothing' },
            { id: 'cat2', name: 'Accessories', slug: 'accessories' },
          ],
        });
      }
      if (url.includes('/api/v1/admin/content/products')) {
        return Promise.resolve({
          data: [
            {
              id: 'p1',
              name: 'Silk Shirt',
              slug: 'silk-shirt',
              category: 'cat1',
              base_price: 1250,
              currency: 'TRY',
              status: 'published',
              cover_image: 'media_123',
            },
            {
              id: 'p2',
              name: 'Leather Belt',
              slug: 'leather-belt',
              category: 'cat2',
              base_price: 450,
              currency: 'TRY',
              status: 'draft',
              cover_image: 'https://example.com/belt.png',
            },
            {
              id: 'p3',
              name: 'Wool Scarf',
              slug: 'wool-scarf',
              category: 'cat1',
              base_price: 300,
              currency: 'TRY',
              status: 'published',
              cover_image: null,
            },
          ],
        });
      }
      if (url.includes('/api/v1/admin/content/variants')) {
        return Promise.resolve({
          data: [
            { id: 'v1', product: 'p1', sku: 'SS-BLK-S' },
            { id: 'v2', product: 'p1', sku: 'SS-BLK-M' },
            { id: 'v3', product: 'p2', sku: 'LB-BRN-34' },
          ],
        });
      }
      if (url.includes('/api/v1/admin/commerce/stock')) {
        return Promise.resolve({
          data: [
            { variant_id: 'v1', available: 2, reserved: 0 },
            { variant_id: 'v2', available: 10, reserved: 1 },
            { variant_id: 'v3', available: 0, reserved: 0 },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });
  });

  afterEach(() => {
    auth.user = null;
    auth.permissions = [];
  });

  it('renders products list with category, price, variant count and status', async () => {
    render(ProductsView);

    expect(await screen.findByText('Silk Shirt')).toBeTruthy();
    expect(screen.getByText('Leather Belt')).toBeTruthy();
    expect(screen.getByText('Wool Scarf')).toBeTruthy();

    expect(screen.getByText('silk-shirt')).toBeTruthy();
    expect(screen.getByText('leather-belt')).toBeTruthy();

    // Categories
    expect(screen.getAllByText('Clothing').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Accessories').length).toBeGreaterThanOrEqual(1);

    // Status badges
    expect(screen.getAllByText('Yayında').length).toBe(2);
    expect(screen.getByText('Taslak')).toBeTruthy();
  });

  it('requests the first page with limit 20 and loads variants only for visible products', async () => {
    render(ProductsView);
    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    const urls = (api.get as any).mock.calls.map((c: any[]) => String(c[0]));
    expect(urls.some((u: string) => u.includes('/products?limit=20&offset=0'))).toBe(true);
    const variantUrl = urls.find((u: string) => u.includes('/content/variants'));
    expect(variantUrl).toContain('filter[product][in]=');
    expect(variantUrl).toContain('p1');
    expect(variantUrl).toContain('p3');
  });

  it('sends the search term to the server (debounced) and resets to page 0', async () => {
    render(ProductsView);
    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'belt' } });

    await waitFor(
      () => {
        const urls = (api.get as any).mock.calls.map((c: any[]) => String(c[0]));
        expect(urls.some((u: string) => u.includes('/products?') && u.includes('search=belt') && u.includes('offset=0'))).toBe(true);
      },
      { timeout: 2000 },
    );
  });

  it('goes to the next page with offset 20 when more products exist', async () => {
    const base = (api.get as any).getMockImplementation();
    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/products')) {
        return base(url).then((res: any) => ({ ...res, metadata: { has_more: true } }));
      }
      return base(url);
    });
    render(ProductsView);
    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    const next = screen.getByRole('button', { name: t('products.pagination.next') });
    expect((next as HTMLButtonElement).disabled).toBe(false);
    await fireEvent.click(next);

    await waitFor(() => {
      const urls = (api.get as any).mock.calls.map((c: any[]) => String(c[0]));
      expect(urls.some((u: string) => u.includes('/products?limit=20&offset=20'))).toBe(true);
    });
  });

  it('displays low-stock badge and out-of-stock badge correctly', async () => {
    render(ProductsView);

    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    // p1 has v1 with 2 available (<= 3), so 'Düşük' badge should show
    expect(screen.getByText('Düşük')).toBeTruthy();

    // p2 has v3 with 0 available, and p3 has no variants (0 total), so 'Stok yok' should show
    expect(screen.getAllByText('Stok yok').length).toBeGreaterThanOrEqual(1);
  });

  it('navigates to product editor when clicking a product row', async () => {
    render(ProductsView);

    const productRowText = await screen.findByText('Silk Shirt');
    const row = productRowText.closest('tr');
    expect(row).toBeTruthy();

    await fireEvent.click(row!);
    expect(router.navigate).toHaveBeenCalledWith('/commerce/products/p1');
  });

  it('navigates to new product editor when clicking the new product button', async () => {
    render(ProductsView);

    const newBtn = await screen.findByTestId('new-product-button');
    expect(newBtn).toBeTruthy();

    await fireEvent.click(newBtn);
    expect(router.navigate).toHaveBeenCalledWith('/commerce/products/new');
  });
});
