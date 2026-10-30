// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import RolesView from './RolesView.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn().mockResolvedValue({ data: { id: 'moderator', name: 'Moderator' } }),
    delete: vi.fn().mockResolvedValue({ data: { deleted: true } }),
  },
}));

describe('RolesView', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders roles table and handles creating a new custom role', async () => {
    (api.get as any).mockResolvedValue({
      data: [
        {
          id: 'moderator',
          name: 'Content Moderator',
          description: 'Can manage comments',
          permissions: [{ resource: 'collection:comments', action: '*' }],
        },
      ],
    });

    render(RolesView);

    // Wait for table to load
    await waitFor(() => {
      expect(screen.getByText('Content Moderator')).toBeTruthy();
      expect(screen.getByText('moderator')).toBeTruthy();
      expect(screen.getByText('collection:comments:*')).toBeTruthy();
    });

    // Open create modal
    const createBtn = screen.getByText('Create Role');
    await fireEvent.click(createBtn);

    // Fill inputs
    const idInput = screen.getByPlaceholderText(/e\.g\. moderator/i);
    await fireEvent.input(idInput, { target: { value: 'store_manager' } });

    const nameInput = screen.getByPlaceholderText(/e\.g\. Content Moderator/i);
    await fireEvent.input(nameInput, { target: { value: 'Store Manager' } });

    // Save role
    const saveBtn = screen.getByRole('button', { name: /Save Role/i });
    await fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith(
        '/api/v1/admin/roles',
        expect.objectContaining({
          id: 'store_manager',
          name: 'Store Manager',
        })
      );
    });
  });
});
