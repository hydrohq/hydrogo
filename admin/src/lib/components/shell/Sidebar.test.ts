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
    schema.schema = { platform: true, modules: {}, resources: [] } as any;
    router.navigate('/console');
    auth.user = { role: 'admin', email: 'admin@example.com' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.getByTitle(t('nav.console'))).toBeTruthy();
    expect(screen.queryByTitle('Pages')).toBeNull();
    expect(screen.queryByTitle('Quick Create')).toBeNull();
  });

  it('hides console and subscription items when platform is false', () => {
    // Arrange
    schema.schema = { platform: false, modules: { pages: true }, resources: [] } as any;
    auth.user = { role: 'admin', email: 'admin@example.com' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.queryByTitle(t('nav.console'))).toBeNull();
    expect(screen.queryByTitle('Abonelik & Ödeme')).toBeNull();
  });

  it('hides console and subscription items before schema is loaded (prevents flash)', () => {
    // Arrange
    schema.schema = null as any;
    auth.user = { role: 'admin', email: 'admin@example.com' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.queryByTitle(t('nav.console'))).toBeNull();
    expect(screen.queryByTitle('Abonelik & Ödeme')).toBeNull();
  });

  it('shows console and subscription items when platform is true', () => {
    // Arrange
    schema.schema = { platform: true, modules: { pages: true }, resources: [] } as any;
    auth.user = { role: 'admin', email: 'admin@example.com' } as any;

    // Act
    render(Sidebar, { props: {} });

    // Assert
    expect(screen.getByTitle(t('nav.console'))).toBeTruthy();
    expect(screen.getByTitle('Abonelik & Ödeme')).toBeTruthy();
  });

  it('gates subscription link in user menu on platform mode', async () => {
    // Arrange - platform false
    schema.schema = { platform: false, modules: {}, resources: [] } as any;
    auth.user = { role: 'admin', email: 'ada@example.com' } as any;
    const { unmount } = render(Sidebar, { props: {} });
    await fireEvent.click(screen.getByText('ada@example.com'));

    // Assert - hidden
    expect(screen.queryByText('Abonelik & Ödeme')).toBeNull();
    unmount();

    // Arrange - platform true
    schema.schema = { platform: true, modules: {}, resources: [] } as any;
    render(Sidebar, { props: {} });
    await fireEvent.click(screen.getByText('ada@example.com'));

    // Assert - shown in both sidebar navigation and user menu
    expect(screen.getAllByText('Abonelik & Ödeme').length).toBe(2);
  });
});
