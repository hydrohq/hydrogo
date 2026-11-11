// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import { auth } from '$lib/stores/auth.svelte';
import { i18n } from '$lib/i18n/index.svelte';
import { confirmAction } from '$lib/confirm';
import VariantsTab from './VariantsTab.svelte';

vi.mock('$lib/confirm', () => ({
  confirmAction: vi.fn().mockResolvedValue(true),
}));

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn().mockResolvedValue({ data: { id: 'v-new' } }),
    delete: vi.fn().mockResolvedValue({}),
  },
  ApiClientError: class ApiClientError extends Error {
    requestId = '';
    status = 500;
  },
}));

describe('VariantsTab', () => {
  const sampleProduct = {
    id: 'p1',
    name: 'Silk Shirt',
    slug: 'silk-shirt',
  };

  beforeEach(() => {
    vi.clearAllMocks();
    i18n.locale = 'tr';
    auth.user = { role: 'admin' } as any;
    auth.permissions = ['admin.commerce.write'];

    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/variants')) {
        return Promise.resolve({
          data: [
            { id: 'v1', product: 'p1', sku: 'SS-BLK-S', color: 'Black', size: 'S' },
            { id: 'v2', product: 'p1', sku: 'SS-RED-M', color: 'Red', size: 'M' },
          ],
        });
      }
      if (url.includes('/api/v1/admin/commerce/stock')) {
        return Promise.resolve({
          data: [
            { variant_id: 'v1', available: 5, reserved: 2 },
            { variant_id: 'v2', available: 0, reserved: 0 },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });

    (api.post as any).mockResolvedValue({ data: { id: 'v-new' } });
  });

  afterEach(() => {
    i18n.locale = 'en';
    auth.user = null;
    auth.permissions = [];
  });

  it('renders matrix and chips from existing variants', async () => {
    render(VariantsTab, { props: { product: sampleProduct } });

    // Chips rendered
    expect((await screen.findAllByText('Black')).length).toBeGreaterThan(0);
    expect(screen.getAllByText('Red').length).toBeGreaterThan(0);
    expect(screen.getAllByText('S').length).toBeGreaterThan(0);
    expect(screen.getAllByText('M').length).toBeGreaterThan(0);

    // v1: Black S input has value 5 and '2 rezerve'
    const inputV1 = screen.getByLabelText('Silk Shirt Black S') as HTMLInputElement;
    expect(inputV1.value).toBe('5');
    expect(screen.getByText('2 rezerve')).toBeTruthy();

    // v2: Red M input has value 0 and red background
    const inputV2 = screen.getByLabelText('Silk Shirt Red M') as HTMLInputElement;
    expect(inputV2.value).toBe('0');
    expect(inputV2.className).toContain('bg-red');

    // Missing combos (Black M, Red S) render '-'
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThanOrEqual(2);
  });

  it('editing a cell posts to /api/v1/admin/commerce/stock with the right body', async () => {
    render(VariantsTab, { props: { product: sampleProduct } });

    const inputV1 = (await screen.findByLabelText('Silk Shirt Black S')) as HTMLInputElement;
    expect(inputV1.value).toBe('5');

    await fireEvent.input(inputV1, { target: { value: '12' } });
    await fireEvent.blur(inputV1);

    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/commerce/stock', {
      variant_id: 'v1',
      available: 12,
    });
  });

  it('executes create-missing flow posting variants then stock with right bodies', async () => {
    render(VariantsTab, { props: { product: sampleProduct } });

    // Missing combos: Black M, Red S (total 2)
    const createBtn = await screen.findByRole('button', {
      name: /Eksik varyantları oluştur \(2\)/,
    });
    expect(createBtn).toBeTruthy();

    await fireEvent.click(createBtn);

    // Verify variants created with generated SKU
    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/content/variants', {
      product: 'p1',
      sku: 'SILK-SHIRT-BLACK-M',
      color: 'Black',
      size: 'M',
    });
    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/content/variants', {
      product: 'p1',
      sku: 'SILK-SHIRT-RED-S',
      color: 'Red',
      size: 'S',
    });

    // Verify initial stock post
    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/commerce/stock', {
      variant_id: 'v-new',
      available: 0,
    });
  });

  it('blocks deletion of chip when variants have reserved stock > 0', async () => {
    render(VariantsTab, { props: { product: sampleProduct } });

    // Black has variant v1 with reserved: 2
    const removeBlackBtn = await screen.findByLabelText('Kaldır Black');
    await fireEvent.click(removeBlackBtn);

    // Confirmation dialog should NOT be called
    expect(confirmAction).not.toHaveBeenCalled();
    // DELETE api should NOT be called
    expect(api.delete).not.toHaveBeenCalled();
    // Refusal message shown
    expect(screen.getByRole('alert')).toBeTruthy();
    expect(screen.getByText(/Silinemez: Bazı varyantlarda 2 adet rezerve stok bulunmaktadır\./)).toBeTruthy();
  });

  it('allows deletion of chip when reserved is 0 after confirmation', async () => {
    (confirmAction as any).mockResolvedValue(true);
    render(VariantsTab, { props: { product: sampleProduct } });

    // Red has variant v2 with reserved: 0
    const removeRedBtn = await screen.findByLabelText('Kaldır Red');
    await fireEvent.click(removeRedBtn);

    // Confirmation dialog should be called
    expect(confirmAction).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Varyantları Sil',
        variant: 'destructive',
      })
    );

    // DELETE api should be called for v2
    expect(api.delete).toHaveBeenCalledWith('/api/v1/admin/content/variants/v2');
  });
});
