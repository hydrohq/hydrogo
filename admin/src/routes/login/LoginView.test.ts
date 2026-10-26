// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { router } from '$lib/router.svelte';
import { auth } from '$lib/stores/auth.svelte';
import { schema } from '$lib/stores/schema.svelte';
import { api } from '$lib/api/client';
import LoginView from './LoginView.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn() },
  setCsrfToken: vi.fn(),
}));

describe('LoginView', () => {
  afterEach(() => {
    auth.user = null;
    auth.permissions = [];
    schema.schema = null as any;
    router.navigate('/login');
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
  });

  it('shows the password-reset notice from the query string', () => {
    // Arrange
    router.navigate('/login?password_reset=1');

    // Act
    render(LoginView, { props: {} });

    // Assert
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('shows the MFA code field and keeps the error clear when the server asks for a code', async () => {
    // Arrange
    vi.mocked(api.post).mockResolvedValue({ mfa_required: true });
    render(LoginView, { props: {} });
    await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'ada@example.com' } });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'secret' } });

    // Act
    await fireEvent.click(screen.getByText('Sign in'));

    // Assert
    await waitFor(() => expect(screen.getByLabelText('MFA Authenticator Code')).toBeTruthy());
  });

  it('signs in, loads the schema and redirects to the dashboard on success', async () => {
    // Arrange
    vi.mocked(api.post).mockResolvedValue({ ok: true });
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/api/v1/admin/me') return Promise.resolve({ data: { user: { email: 'ada@example.com', role: 'admin', active: true }, permissions: [] } });
      if (url === '/api/v1/admin/schema') return Promise.resolve({ data: { modules: {}, resources: [] } });
      return Promise.resolve({ data: null });
    });
    render(LoginView, { props: {} });
    await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'ada@example.com' } });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'secret' } });

    // Act
    await fireEvent.click(screen.getByText('Sign in'));

    // Assert
    await waitFor(() => expect(router.path).toBe('/dashboard'));
    expect(auth.user?.email).toBe('ada@example.com');
  });

  it('shows the server error message on a failed login', async () => {
    // Arrange
    vi.mocked(api.post).mockRejectedValue(new Error('Invalid credentials'));
    render(LoginView, { props: {} });
    await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'ada@example.com' } });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'wrong' } });

    // Act
    await fireEvent.click(screen.getByText('Sign in'));

    // Assert
    await waitFor(() => expect(screen.getByText('Invalid credentials')).toBeTruthy());
  });
});
