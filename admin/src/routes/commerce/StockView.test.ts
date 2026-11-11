// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { t } from '$lib/i18n/index.svelte';
import { api } from '$lib/api/client';
import { auth } from '$lib/stores/auth.svelte';
import StockView from './StockView.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn().mockResolvedValue({}),
  },
  ApiClientError: class ApiClientError extends Error {
    requestId = '';
  },
}));

describe('StockView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    auth.user = { role: 'admin' } as any;
    auth.permissions = ['admin.commerce.write'];

    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/products')) {
        return Promise.resolve({
          data: [{ id: 'p1', name: 'Silk Shirt', slug: 'silk-shirt' }],
          metadata: { limit: 10, offset: 0, total: 1, has_more: false },
        });
      }
      if (url.includes('/api/v1/admin/content/variants')) {
        return Promise.resolve({
          data: [
            { id: 'v1', product: 'p1', sku: 'SS-BLK-S', color: 'Black', size: 'S' },
            { id: 'v2', product: 'p1', sku: 'SS-BLK-M', color: 'Black', size: 'M' },
            { id: 'v3', product: 'p1', sku: 'SS-RED-S', color: 'Red', size: 'S' },
          ],
        });
      }
      if (url.includes('/api/v1/admin/commerce/stock')) {
        return Promise.resolve({
          data: [
            { variant_id: 'v1', available: 5, reserved: 2 },
            { variant_id: 'v2', available: 0, reserved: 0 },
            { variant_id: 'v3', available: 2, reserved: 1 },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });

    (api.post as any).mockResolvedValue({ data: { success: true } });
  });

  afterEach(() => {
    auth.user = null;
    auth.permissions = [];
  });

  it('renders matrix from mocked responses', async () => {
    render(StockView);

    expect(await screen.findByText('Silk Shirt')).toBeTruthy();
    expect(screen.getByText('Black')).toBeTruthy();
    expect(screen.getByText('Red')).toBeTruthy();
    expect(screen.getByText('S')).toBeTruthy();
    expect(screen.getByText('M')).toBeTruthy();

    // v1: Black S has 5 available and 2 reserved
    const inputV1 = screen.getByLabelText('Silk Shirt Black S') as HTMLInputElement;
    expect(inputV1.value).toBe('5');
    expect(screen.getByText('2 rezerve')).toBeTruthy();

    // Red M combination has no variant, renders dash
    expect(screen.getByText('-')).toBeTruthy();
  });

  it('requests the first page with limit 10, offset 0, and filters variants and stock by visible products', async () => {
    render(StockView);
    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    const urls = (api.get as any).mock.calls.map((c: any[]) => String(c[0]));
    expect(
      urls.some(
        (u: string) =>
          u.includes('/api/v1/admin/content/products') &&
          u.includes('limit=10') &&
          u.includes('offset=0') &&
          u.includes('sort=name')
      )
    ).toBe(true);

    const variantUrl = urls.find((u: string) => u.includes('/content/variants'));
    expect(variantUrl).toContain('filter[product][in]=p1');

    const stockUrl = urls.find((u: string) => u.includes('/commerce/stock'));
    expect(stockUrl).toContain('variant_ids=');
    expect(decodeURIComponent(stockUrl!)).toContain('v1,v2,v3');
  });

  it('sends the search term to the server (debounced) and resets to page 0', async () => {
    render(StockView);
    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'shirt' } });

    await waitFor(
      () => {
        const urls = (api.get as any).mock.calls.map((c: any[]) => String(c[0]));
        expect(
          urls.some(
            (u: string) =>
              u.includes('/products?') &&
              u.includes('search=shirt') &&
              u.includes('offset=0') &&
              u.includes('sort=name')
          )
        ).toBe(true);
      },
      { timeout: 2000 }
    );
  });

  it('goes to the next page with offset 10 when more products exist', async () => {
    const base = (api.get as any).getMockImplementation();
    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/products')) {
        return base(url).then((res: any) => ({ ...res, metadata: { has_more: true } }));
      }
      return base(url);
    });

    render(StockView);
    expect(await screen.findByText('Silk Shirt')).toBeTruthy();

    const next = screen.getByRole('button', { name: t('stock.pagination.next') });
    expect((next as HTMLButtonElement).disabled).toBe(false);
    await fireEvent.click(next);

    await waitFor(() => {
      const urls = (api.get as any).mock.calls.map((c: any[]) => String(c[0]));
      expect(
        urls.some(
          (u: string) =>
            u.includes('/products?') &&
            u.includes('limit=10') &&
            u.includes('offset=10') &&
            u.includes('sort=name')
        )
      ).toBe(true);
    });
  });

  it('editing a cell posts to /api/v1/admin/commerce/stock with the right body', async () => {
    render(StockView);

    const inputV1 = (await screen.findByLabelText('Silk Shirt Black S')) as HTMLInputElement;
    expect(inputV1.value).toBe('5');

    await fireEvent.input(inputV1, { target: { value: '12' } });
    await fireEvent.blur(inputV1);

    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/commerce/stock', {
      variant_id: 'v1',
      available: 12,
    });
  });

  it('applies 0-stock styling for out-of-stock variants', async () => {
    render(StockView);

    const inputV2 = (await screen.findByLabelText('Silk Shirt Black M')) as HTMLInputElement;
    expect(inputV2.value).toBe('0');
    expect(inputV2.className).toContain('bg-red');
  });

  it('disables inputs when user has no write permission', async () => {
    auth.user = { role: 'viewer' } as any;
    auth.permissions = [];

    render(StockView);

    const inputV1 = (await screen.findByLabelText('Silk Shirt Black S')) as HTMLInputElement;
    expect(inputV1.disabled).toBe(true);
  });
});
