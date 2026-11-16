// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import { i18n } from '$lib/i18n/index.svelte';
import ContentTypesView from './ContentTypesView.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn().mockResolvedValue({ data: { id: 'test' } }),
    put: vi.fn().mockResolvedValue({ data: { id: 'faq' } }),
    delete: vi.fn().mockResolvedValue({ data: { deleted: true } }),
  },
}));

describe('ContentTypesView', () => {
  beforeEach(() => {
    i18n.locale = 'en';
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders content types list and creates a new content type via visual builder', async () => {
    (api.get as any).mockResolvedValue({
      data: [
        {
          id: 'faq',
          label: 'FAQ',
          description: 'Frequently asked questions',
          public: true,
          fields: [{ id: 'question', label: 'Question', kind: 'text' }],
        },
      ],
    });

    render(ContentTypesView, { props: { mode: 'list' } });

    await waitFor(() => {
      expect(screen.getByText('FAQ')).toBeTruthy();
      expect(screen.getByText('/api/v1/content/faq')).toBeTruthy();
    });

    const newBtn = screen.getByText('New Object');
    await fireEvent.click(newBtn);

    const labelInput = screen.getByPlaceholderText(/Testimonials/i);
    await fireEvent.input(labelInput, { target: { value: 'Product' } });

    const numberPaletteBtn = screen.getByText('+ number');
    await fireEvent.click(numberPaletteBtn);

    const submitBtn = screen.getByRole('button', { name: /Create New Object/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith(
        '/api/v1/admin/content-types',
        expect.objectContaining({
          id: 'product',
          label: 'Product',
          public: true,
          drafts: true,
          fields: expect.arrayContaining([
            expect.objectContaining({ kind: 'text' }),
            expect.objectContaining({ kind: 'number' }),
          ]),
        })
      );
    });
  });

  it('automatically opens creation modal when mode is new', async () => {
    (api.get as any).mockResolvedValue({ data: [] });

    render(ContentTypesView, { props: { mode: 'new' } });

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/Testimonials/i)).toBeTruthy();
      expect(screen.getByText('+ relation')).toBeTruthy();
    });
  });

  it('shows accurate access badge for write-only public types and supports editing existing schemas', async () => {
    (api.get as any).mockImplementation((url: string) => {
      if (url === '/api/v1/admin/content-types') {
        return Promise.resolve({
          data: [
            {
              id: 'faq',
              label: 'FAQ',
              public: true,
              group: 'Content',
              fields: [{ id: 'question', label: 'Question', kind: 'text', required: true }],
            },
            {
              id: 'leads',
              label: 'Leads',
              public: true,
              group: 'Support',
              access_rules: { list: 'false', read: 'false', create: 'public' },
              fields: [{ id: 'email', label: 'Email', kind: 'text', required: true }],
            },
          ],
        });
      }
      return Promise.resolve({ data: {} });
    });

    render(ContentTypesView, { props: { mode: 'list' } });

    await waitFor(() => {
      expect(screen.getByText('FAQ')).toBeTruthy();
      expect(screen.getByText('Form intake only')).toBeTruthy();
      expect(screen.getByText('Public API')).toBeTruthy();
    });

    const editBtns = screen.getAllByRole('button', { name: 'Edit schema' });
    await fireEvent.click(editBtns[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit Object: FAQ')).toBeTruthy();
    });

    const saveBtn = screen.getByRole('button', { name: 'Save changes' });
    await fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(api.put).toHaveBeenCalledWith(
        '/api/v1/admin/content-types/faq',
        expect.objectContaining({
          id: 'faq',
          label: 'FAQ',
          fields: expect.arrayContaining([
            expect.objectContaining({ id: 'question', kind: 'text', required: true }),
          ]),
        })
      );
    });
  });
});
