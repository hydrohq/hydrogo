// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import { schema } from '$lib/stores/schema.svelte';
import { i18n } from '$lib/i18n/index.svelte';
import RelationField from './RelationField.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
  },
}));

describe('RelationField', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    i18n.locale = 'en';
    schema.schema = {
      version: 1,
      resources: [
        {
          id: 'products',
          label: 'Products',
          singular: 'Product',
          plural: 'Products',
          title_field: 'name',
          api_path: '/api/v1/admin/content/products',
          fields: [
            { id: 'name', label: 'Name', kind: 'text' },
            { id: 'slug', label: 'Slug', kind: 'text' },
          ],
        },
      ],
      modules: {},
      locales: { enabled: false, default: 'en', all: ['en'] },
    } as any;

    (api.get as any).mockImplementation((_url: string, opts?: { params?: Record<string, unknown> }) => {
      const search = String(opts?.params?.search ?? '').toLowerCase();
      const all = [
        { id: 'p1', name: 'Classic Belt', slug: 'classic-belt', sku: 'BLT-01', cover_image: '/media/belt.jpg' },
        { id: 'p2', name: 'Brass Buckle', slug: 'brass-buckle', sku: 'BCK-02' },
        { id: 'p3', name: 'Gift Box', slug: 'gift-box', sku: 'BOX-03' },
      ];
      return Promise.resolve({
        data: search ? all.filter((item) => item.name.toLowerCase().includes(search)) : all,
      });
    });
  });

  afterEach(() => {
    schema.schema = null;
  });

  it('picks and removes records by slug into comma-separated string', async () => {
    let current = 'classic-belt';
    const { rerender } = render(RelationField, {
      props: {
        id: 'related_products',
        label: 'Related products',
        target: 'products',
        many: true,
        valueField: 'slug',
        get value() {
          return current;
        },
        set value(next: string) {
          current = next;
        },
      },
    });

    await waitFor(() => {
      expect(screen.getByText('Brass Buckle')).toBeTruthy();
      expect(screen.getByText('Classic Belt')).toBeTruthy();
    });

    await fireEvent.click(screen.getByRole('option', { name: /Brass Buckle/i }));
    await waitFor(() => {
      expect(current).toBe('classic-belt, brass-buckle');
    });

    await rerender({
      id: 'related_products',
      label: 'Related products',
      target: 'products',
      many: true,
      valueField: 'slug',
      value: current,
    });

    const removeBtn = screen.getByRole('button', { name: 'Remove Classic Belt' });
    await fireEvent.click(removeBtn);
    await waitFor(() => {
      expect(current).toBe('brass-buckle');
    });
  });

  it('filters options via search input and supports single id relation', async () => {
    let current = '';
    render(RelationField, {
      props: {
        id: 'product',
        label: 'Product',
        target: 'products',
        many: false,
        valueField: 'id',
        get value() {
          return current;
        },
        set value(next: string) {
          current = next;
        },
      },
    });

    await waitFor(() => {
      expect(screen.getByText('Gift Box')).toBeTruthy();
    });

    const searchInput = screen.getByPlaceholderText('Search Products...');
    await fireEvent.input(searchInput, { target: { value: 'gift' } });

    await waitFor(() => {
      expect(api.get).toHaveBeenCalledWith(
        '/api/v1/admin/content/products',
        expect.objectContaining({ params: expect.objectContaining({ search: 'gift' }) })
      );
    });

    await fireEvent.click(screen.getByRole('option', { name: /Gift Box/i }));
    await waitFor(() => {
      expect(current).toBe('p3');
    });
  });
});
