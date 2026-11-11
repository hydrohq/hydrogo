// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
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
