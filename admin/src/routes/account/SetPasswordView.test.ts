// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { router } from '$lib/router.svelte';
import { api, ApiClientError } from '$lib/api/client';
import SetPasswordView from './SetPasswordView.svelte';

vi.mock('$lib/api/client', async () => {
  class ApiClientError extends Error {
    status: number;
    constructor(error: { message?: string }, status: number) {
      super(error.message || 'API request failed');
      this.status = status;
    }
  }
  return { api: { post: vi.fn() }, ApiClientError };
});

describe('SetPasswordView', () => {
  afterEach(() => {
    router.navigate('/account/reset');
    vi.mocked(api.post).mockReset();
  });

  it('shows an invalid-link message when there is no token', () => {
    // Arrange
    router.navigate('/account/reset');

    // Act
    render(SetPasswordView, { props: { mode: 'reset' } });

    // Assert
    expect(screen.getByRole('alert')).toBeTruthy();
    expect(screen.queryByLabelText('Password')).toBeNull();
  });

  it('rejects a too-short password without calling the API', async () => {
    // Arrange
    router.navigate('/account/reset?token=abc');
    render(SetPasswordView, { props: { mode: 'reset' } });
    await fireEvent.input(screen.getByLabelText('New password'), { target: { value: 'short' } });
    await fireEvent.input(screen.getByLabelText('Repeat the password'), { target: { value: 'short' } });

    // Act
    await fireEvent.click(screen.getByText('Save password'));

    // Assert
    expect(api.post).not.toHaveBeenCalled();
    expect(screen.getByRole('alert')).toBeTruthy();
  });

  it('rejects mismatched passwords without calling the API', async () => {
    // Arrange
    router.navigate('/account/reset?token=abc');
    render(SetPasswordView, { props: { mode: 'reset' } });
    await fireEvent.input(screen.getByLabelText('New password'), { target: { value: 'longenough1' } });
    await fireEvent.input(screen.getByLabelText('Repeat the password'), { target: { value: 'different1' } });

    // Act
    await fireEvent.click(screen.getByText('Save password'));

    // Assert
    expect(api.post).not.toHaveBeenCalled();
  });

  it('submits the reset and redirects to login on success', async () => {
    // Arrange
    vi.mocked(api.post).mockResolvedValue({});
    const assign = vi.fn();
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', { value: { ...originalLocation, assign }, configurable: true });
    router.navigate('/account/reset?token=abc123');
    render(SetPasswordView, { props: { mode: 'reset' } });
    await fireEvent.input(screen.getByLabelText('New password'), { target: { value: 'longenough1' } });
    await fireEvent.input(screen.getByLabelText('Repeat the password'), { target: { value: 'longenough1' } });

    // Act
    await fireEvent.click(screen.getByText('Save password'));

    // Assert
    await waitFor(() => expect(assign).toHaveBeenCalledWith('/admin/login?password_reset=1'));
    Object.defineProperty(window, 'location', { value: originalLocation, configurable: true });
  });

  it('shows an expired-link message on a 401/404 response', async () => {
    // Arrange
    vi.mocked(api.post).mockRejectedValue(new ApiClientError({ message: 'expired' } as any, 401));
    router.navigate('/account/reset?token=abc123');
    render(SetPasswordView, { props: { mode: 'reset' } });
    await fireEvent.input(screen.getByLabelText('New password'), { target: { value: 'longenough1' } });
    await fireEvent.input(screen.getByLabelText('Repeat the password'), { target: { value: 'longenough1' } });

    // Act
    await fireEvent.click(screen.getByText('Save password'));

    // Assert
    await waitFor(() => expect(screen.getByRole('alert').textContent).toMatch(/link/i));
  });
});
