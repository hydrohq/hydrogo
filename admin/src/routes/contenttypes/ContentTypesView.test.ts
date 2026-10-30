// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import ContentTypesView from './ContentTypesView.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn().mockResolvedValue({ data: { id: 'test' } }),
    delete: vi.fn().mockResolvedValue({ data: { deleted: true } }),
  },
}));

describe('ContentTypesView', () => {
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
          fields: [{ id: 'question', label: 'Question', kind: 'text' }],
        },
      ],
    });

    render(ContentTypesView, { props: { mode: 'list' } });

    // Wait for content types to load
    await waitFor(() => {
      expect(screen.getByText('FAQ')).toBeTruthy();
      expect(screen.getByText('/api/v1/content/faq')).toBeTruthy();
    });

    // Open creation modal
    const newBtn = screen.getByText('New Object');
    await fireEvent.click(newBtn);

    // Enter label and ID
    const labelInput = screen.getByPlaceholderText(/Testimonials/i);
    await fireEvent.input(labelInput, { target: { value: 'Product' } });

    // Add field from Palette
    const numberPaletteBtn = screen.getByText('+ number');
    await fireEvent.click(numberPaletteBtn);

    // Save
    const submitBtn = screen.getByRole('button', { name: /Create New Object/i });
    await fireEvent.click(submitBtn);

    // Check api.post called with correct payload
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
});
