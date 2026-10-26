// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { auth } from '$lib/stores/auth.svelte';
import { api } from '$lib/api/client';
import AccountView from './AccountView.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn(), delete: vi.fn() },
}));

describe('AccountView', () => {
  afterEach(() => {
    auth.user = null;
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
    vi.mocked(api.delete).mockReset();
  });

  it('shows the signed-in user on the profile tab', () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });

    // Act
    render(AccountView, { props: {} });

    // Assert
    expect(screen.getByRole('heading', { name: 'ada@example.com' })).toBeTruthy();
  });

  it('rejects a mismatched new password without calling the API', async () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    render(AccountView, { props: {} });
    await fireEvent.click(screen.getByText('Password'));

    // Act
    await fireEvent.input(screen.getByLabelText('Current Password'), { target: { value: 'oldpass1' } });
    await fireEvent.input(screen.getByLabelText('New Password'), { target: { value: 'newpassword1' } });
    await fireEvent.input(screen.getByLabelText('Confirm New Password'), { target: { value: 'different1' } });
    await fireEvent.click(screen.getByText('Save Password'));

    // Assert
    expect(api.post).not.toHaveBeenCalled();
  });

  it('changes the password successfully', async () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    vi.mocked(api.post).mockResolvedValue({});
    render(AccountView, { props: {} });
    await fireEvent.click(screen.getByText('Password'));
    await fireEvent.input(screen.getByLabelText('Current Password'), { target: { value: 'oldpass1' } });
    await fireEvent.input(screen.getByLabelText('New Password'), { target: { value: 'newpassword1' } });
    await fireEvent.input(screen.getByLabelText('Confirm New Password'), { target: { value: 'newpassword1' } });

    // Act
    await fireEvent.click(screen.getByText('Save Password'));

    // Assert
    await waitFor(() => expect(screen.getByText('Password updated successfully.')).toBeTruthy());
    expect(api.post).toHaveBeenCalledWith('/auth/me/credentials', {
      current_password: 'oldpass1',
      new_password: 'newpassword1',
    });
  });

  it('loads and lists active sessions', async () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/auth/me/sessions') return Promise.resolve({ data: [{ id: 's1', ip_address: '1.2.3.4', current: true }] });
      return Promise.resolve({ data: [] });
    });
    render(AccountView, { props: {} });

    // Act
    await fireEvent.click(screen.getByText(/Active Sessions/));

    // Assert
    await waitFor(() => expect(screen.getByText('1.2.3.4', { exact: false })).toBeTruthy());
  });

  it('creates a personal token and shows it once for copying', async () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    vi.mocked(api.post).mockResolvedValue({ data: { token: 'hg_abc123' } });
    render(AccountView, { props: {} });
    await fireEvent.click(screen.getByText(/Personal Tokens/));
    await fireEvent.click(screen.getByText('Generate New Token'));
    await fireEvent.input(screen.getByLabelText('Token Name'), { target: { value: 'CI token' } });

    // Act
    await fireEvent.click(screen.getByText('Generate Token'));

    // Assert
    await waitFor(() => expect(screen.getByDisplayValue('hg_abc123')).toBeTruthy());
  });
});
