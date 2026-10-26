// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { auth } from '$lib/stores/auth.svelte';
import { api } from '$lib/api/client';
import UsersView from './UsersView.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

function user(over: Partial<any> = {}) {
  return { id: 'u1', email: 'bob@example.com', role: 'editor', active: true, ...over };
}

describe('UsersView', () => {
  afterEach(() => {
    auth.user = null;
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
    vi.mocked(api.delete).mockReset();
  });

  it('lists staff users with their role', async () => {
    // Arrange
    auth.user = { id: 'me', role: 'admin', email: 'admin@example.com' } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/api/v1/admin/users') return Promise.resolve({ data: [user()] });
      return Promise.resolve({ data: [] });
    });

    // Act
    render(UsersView, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('bob@example.com')).toBeTruthy());
  });

  it('invites a new staff member', async () => {
    // Arrange
    auth.user = { id: 'me', role: 'admin', email: 'admin@example.com' } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    vi.mocked(api.post).mockResolvedValue({ delivered: true });
    render(UsersView, { props: {} });
    await waitFor(() => expect(screen.getByText('No staff users found.')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Invite Staff Member'));
    await fireEvent.input(screen.getByLabelText('Email Address'), { target: { value: 'new@example.com' } });
    await fireEvent.click(screen.getByText('Send Invitation'));

    // Assert
    await waitFor(() => expect(api.post).toHaveBeenCalledWith(
      '/api/v1/admin/users/invites',
      'email=new%40example.com&role=editor',
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } },
    ));
  });

  it('shows an invite link instead of a success toast when no mail was delivered', async () => {
    // Arrange
    auth.user = { id: 'me', role: 'admin', email: 'admin@example.com' } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    vi.mocked(api.post).mockResolvedValue({ delivered: false, url: 'https://example.com/invite/abc', email: 'new@example.com' });
    render(UsersView, { props: {} });
    await waitFor(() => expect(screen.getByText('No staff users found.')).toBeTruthy());
    await fireEvent.click(screen.getByText('Invite Staff Member'));
    await fireEvent.input(screen.getByLabelText('Email Address'), { target: { value: 'new@example.com' } });

    // Act
    await fireEvent.click(screen.getByText('Send Invitation'));

    // Assert
    await waitFor(() => expect(screen.getByDisplayValue('https://example.com/invite/abc')).toBeTruthy());
  });

  it('changes a staff member role', async () => {
    // Arrange
    auth.user = { id: 'me', role: 'admin', email: 'admin@example.com' } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/api/v1/admin/users') return Promise.resolve({ data: [user()] });
      return Promise.resolve({ data: [] });
    });
    vi.mocked(api.post).mockResolvedValue({});
    render(UsersView, { props: {} });
    await waitFor(() => expect(screen.getByText('bob@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Role'));
    await fireEvent.click(screen.getByText('Save Role'));

    // Assert
    await waitFor(() => expect(api.post).toHaveBeenCalledWith(
      '/api/v1/admin/users/u1/role',
      'role=editor',
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } },
    ));
  });

  it('disables an active staff member through the confirm dialog', async () => {
    // Arrange
    auth.user = { id: 'me', role: 'admin', email: 'admin@example.com' } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/api/v1/admin/users') return Promise.resolve({ data: [user()] });
      return Promise.resolve({ data: [] });
    });
    vi.mocked(api.post).mockResolvedValue({});
    render(UsersView, { props: {} });
    await waitFor(() => expect(screen.getByText('bob@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Disable'));
    await fireEvent.click(screen.getAllByText('Disable').at(-1)!);

    // Assert
    await waitFor(() => expect(api.post).toHaveBeenCalledWith('/api/v1/admin/users/u1/disable', {}));
  });

  it('creates a system API key and lists it afterwards', async () => {
    // Arrange
    auth.user = { id: 'me', role: 'admin', email: 'admin@example.com' } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/api/v1/admin/api-keys') return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
    vi.mocked(api.post).mockResolvedValue({ data: { key: 'hgk_live_123' } });
    render(UsersView, { props: {} });
    await fireEvent.click(screen.getByText(/System API Keys/));
    await fireEvent.click(screen.getByText('Generate API Key'));
    await fireEvent.input(screen.getByLabelText('Key Name / Service'), { target: { value: 'CI' } });

    // Act
    await fireEvent.click(screen.getByText('Generate Key'));

    // Assert
    await waitFor(() => expect(screen.getByDisplayValue('hgk_live_123')).toBeTruthy());
  });
});
