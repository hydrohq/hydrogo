// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { router } from '$lib/router.svelte';
import { schema } from '$lib/stores/schema.svelte';
import { sidebar } from '$lib/stores/sidebar.svelte';
import { auth } from '$lib/stores/auth.svelte';
import { api } from '$lib/api/client';
import { t } from '$lib/i18n/index.svelte';
import Sidebar from './Sidebar.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn().mockResolvedValue({}) },
}));

describe('Sidebar', () => {
  afterEach(() => {
    schema.schema = null as any;
    auth.user = null;
    auth.permissions = [];
    sidebar.open = true;
    sidebar.mobileOpen = false;
    router.navigate('/dashboard');
  });

  it('hides a module nav item when the module is off', () => {
    // Arrange
    schema.schema = { modules: { pages: false, news: true }, resources: [] } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.queryByTitle('Pages')).toBeNull();
    expect(screen.getByTitle('News')).toBeTruthy();
  });

  it('hides the quick-create button without content/media write permission', () => {
    // Arrange
    auth.user = { role: 'editor' } as any;
    auth.permissions = [];

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.queryByTitle('Quick Create')).toBeNull();
  });

  it('shows the quick-create button for an admin', () => {
    // Arrange
    auth.user = { role: 'admin' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.getByTitle('Quick Create')).toBeTruthy();
  });

  it('shows the signed-in user email and role', () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.getByText('ada@example.com')).toBeTruthy();
  });

  it('logs out through the user menu', async () => {
    // Arrange
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    render(Sidebar, { props: {} });
    await fireEvent.click(screen.getByText('ada@example.com'));

    // Act
    await fireEvent.click(screen.getByText('Log out'));

    // Assert
    expect(api.post).toHaveBeenCalledWith('/auth/logout');
    expect(auth.user).toBeNull();
  });

  it('closes the mobile sidebar when the overlay backdrop is clicked', async () => {
    // Arrange
    sidebar.mobileOpen = true;
    render(Sidebar, { props: {} });

    // Act
    await fireEvent.click(screen.getByLabelText('Close sidebar overlay'));

    // Assert
    expect(sidebar.mobileOpen).toBe(false);
  });

  it('renders dedicated factory navigation and hides standard cms items in console mode', () => {
    // Arrange
    router.navigate('/console');
    auth.user = { role: 'admin', email: 'admin@example.com' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.getByTitle(t('nav.console'))).toBeTruthy();
    expect(screen.queryByTitle('Pages')).toBeNull();
    expect(screen.queryByTitle('Quick Create')).toBeNull();
  });
});
