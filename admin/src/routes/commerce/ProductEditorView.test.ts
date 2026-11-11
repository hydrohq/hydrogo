// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api, ApiClientError } from '$lib/api/client';
import { auth } from '$lib/stores/auth.svelte';
import { router } from '$lib/router.svelte';
import { i18n } from '$lib/i18n/index.svelte';
import ProductEditorView from './ProductEditorView.svelte';
import { slugify } from '$lib/commerce/slugify';

vi.mock('$lib/api/client', () => {
  class MockApiClientError extends Error {
    code = 'ERROR';
    requestId = 'req-123';
    fields?: Record<string, string>;
    status: number;
    constructor(error: any, status = 500) {
      super(error?.message || 'API request failed');
      this.name = 'ApiClientError';
      this.fields = error?.fields;
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

describe('ProductEditorView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    i18n.locale = 'tr';
    auth.user = { role: 'admin' } as any;
    auth.permissions = ['admin.commerce.write', 'admin.content.write'];
    vi.spyOn(router, 'navigate').mockImplementation(() => {});

    // Default GET implementation
    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/categories')) {
        return Promise.resolve({
          data: [
            { id: 'cat1', name: 'Giyim', slug: 'giyim' },
            { id: 'cat2', name: 'Aksesuar', slug: 'aksesuar' },
          ],
        });
      }
      if (url.includes('/api/v1/admin/content/products/p1')) {
        return Promise.resolve({
          data: {
            id: 'p1',
            version: 3,
            name: 'Hakiki Deri Kemer',
            slug: 'hakiki-deri-kemer',
            category: 'cat2',
            base_price: 450,
            currency: 'TRY',
            compare_at_price: 550,
            genuine_leather: true,
            status: 'published',
            description: '<p>Açıklama</p>',
            sku: 'KMR-001',
            seo_title: 'Deri Kemer Satın Al',
            seo_description: 'Kaliteli deri kemer',
          },
        });
      }
      return Promise.resolve({ data: [] });
    });
  });

  afterEach(() => {
    i18n.locale = 'en';
    auth.user = null;
    auth.permissions = [];
  });

  it('slug auto-generation transliterates Turkish characters correctly', () => {
    expect(slugify('Çanta & Şık Örnek İğne')).toBe('canta-sik-ornek-igne');
    expect(slugify('Ağaç Gövdesi Üstünde')).toBe('agac-govdesi-ustunde');
    expect(slugify('Ilık Su ve Çay')).toBe('ilik-su-ve-cay');
  });

  it('loads existing product into form', async () => {
    render(ProductEditorView, { props: { id: 'p1' } });

    await waitFor(() => {
      const nameInput = document.getElementById('product-name') as HTMLInputElement;
      expect(nameInput?.value).toBe('Hakiki Deri Kemer');
    });

    const slugInput = document.getElementById('product-slug') as HTMLInputElement;
    expect(slugInput?.value).toBe('hakiki-deri-kemer');

    const priceInput = document.getElementById('product-price') as HTMLInputElement;
    expect(priceInput?.value).toBe('450');

    // Status badge shows published
    expect(screen.getByText('Yayında')).toBeTruthy();

    // Preview link is present
    const previewLink = screen.getByRole('link', { name: /önizleme/i });
    expect(previewLink).toBeTruthy();
    expect(previewLink.getAttribute('href')).toContain('/urun/hakiki-deri-kemer');

    // Publish/Unpublish toggle button is present
    expect(screen.getByRole('button', { name: /taslağa al/i })).toBeTruthy();
  });

  it('slug auto-generates from name input for new products until manually edited', async () => {
    render(ProductEditorView, { props: { id: 'new' } });

    const nameInput = document.getElementById('product-name') as HTMLInputElement;
    const slugInput = document.getElementById('product-slug') as HTMLInputElement;

    await fireEvent.input(nameInput, { target: { value: 'Özel Çanta' } });
    expect(slugInput.value).toBe('ozel-canta');

    // Manually edit the slug
    await fireEvent.input(slugInput, { target: { value: 'manuel-slug' } });
    expect(slugInput.value).toBe('manuel-slug');

    // Changing the name should no longer overwrite the manually edited slug
    await fireEvent.input(nameInput, { target: { value: 'Yeni Özel Çanta' } });
    expect(slugInput.value).toBe('manuel-slug');
  });

  it('create posts expected body then navigates', async () => {
    (api.post as any).mockResolvedValueOnce({
      data: {
        id: 'p99',
        name: 'Yeni Cüzdan',
        slug: 'yeni-cuzdan',
        base_price: 250,
        currency: 'TRY',
        version: 1,
      },
    });

    render(ProductEditorView, { props: { id: 'new' } });

    const nameInput = document.getElementById('product-name') as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: 'Yeni Cüzdan' } });

    const priceInput = document.getElementById('product-price') as HTMLInputElement;
    await fireEvent.input(priceInput, { target: { value: '250' } });

    const saveButton = screen.getByRole('button', { name: /kaydet/i });
    await fireEvent.click(saveButton);

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith(
        '/api/v1/admin/content/products',
        expect.objectContaining({
          name: 'Yeni Cüzdan',
          slug: 'yeni-cuzdan',
          base_price: 250,
          currency: 'TRY',
        })
      );
    });

    expect(router.navigate).toHaveBeenCalledWith('/commerce/products/p99#varyantlar');
  });

  it('update sends version', async () => {
    (api.put as any).mockResolvedValueOnce({
      data: {
        id: 'p1',
        version: 4,
        name: 'Hakiki Deri Kemer Güncel',
        slug: 'hakiki-deri-kemer',
        base_price: 500,
        currency: 'TRY',
      },
    });

    render(ProductEditorView, { props: { id: 'p1' } });

    await waitFor(() => {
      const nameInput = document.getElementById('product-name') as HTMLInputElement;
      expect(nameInput?.value).toBe('Hakiki Deri Kemer');
    });

    const priceInput = document.getElementById('product-price') as HTMLInputElement;
    await fireEvent.input(priceInput, { target: { value: '500' } });

    const saveButton = screen.getByRole('button', { name: /kaydet/i });
    await fireEvent.click(saveButton);

    await waitFor(() => {
      expect(api.put).toHaveBeenCalledWith(
        '/api/v1/admin/content/products/p1',
        expect.objectContaining({
          version: 3,
          base_price: 500,
        })
      );
    });
  });

  it('422 slug error shown', async () => {
    (api.post as any).mockRejectedValueOnce(
      new ApiClientError(
        {
          code: 'unprocessable_entity',
          request_id: 'req-1',
          message: 'Doğrulama hatası',
          fields: { slug: 'Bu adres zaten kullanımda' },
        } as any,
        422
      )
    );

    render(ProductEditorView, { props: { id: 'new' } });

    const nameInput = document.getElementById('product-name') as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: 'Test Ürünü' } });

    const priceInput = document.getElementById('product-price') as HTMLInputElement;
    await fireEvent.input(priceInput, { target: { value: '100' } });

    const saveButton = screen.getByRole('button', { name: /kaydet/i });
    await fireEvent.click(saveButton);

    expect(await screen.findByText('Bu adres zaten kullanımda')).toBeTruthy();
  });

  it('publish posts workflow', async () => {
    (api.get as any).mockImplementation((url: string) => {
      if (url.includes('/api/v1/admin/content/products/p1')) {
        return Promise.resolve({
          data: {
            id: 'p1',
            version: 2,
            name: 'Taslak Ürün',
            slug: 'taslak-urun',
            base_price: 100,
            currency: 'TRY',
            status: 'draft',
          },
        });
      }
      return Promise.resolve({ data: [] });
    });

    (api.post as any).mockResolvedValueOnce({
      data: {
        id: 'p1',
        version: 3,
        name: 'Taslak Ürün',
        slug: 'taslak-urun',
        status: 'published',
      },
    });

    render(ProductEditorView, { props: { id: 'p1' } });

    await waitFor(() => {
      expect(screen.getByText('Taslak Ürün')).toBeTruthy();
    });

    const publishButton = screen.getByRole('button', { name: /yayınla/i });
    await fireEvent.click(publishButton);

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith('/api/v1/admin/content/products/p1/workflow', {
        target: 'published',
        version: 2,
      });
    });
  });
});
