// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { router } from '$lib/router.svelte';
import { schema } from '$lib/stores/schema.svelte';
import { sidebar } from '$lib/stores/sidebar.svelte';
import { theme } from '$lib/stores/theme.svelte';
import Header from './Header.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn() },
}));

describe('Header', () => {
  afterEach(() => {
    schema.schema = null as any;
    router.navigate('/dashboard');
    sidebar.open = true;
    sidebar.mobileOpen = false;
  });

  it('shows the dashboard title on the dashboard path', () => {
    // Arrange
    router.navigate('/dashboard');

    // Act
    render(Header, { props: {} });

    // Assert
    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeTruthy();
  });

  it('shows a module title derived from the path', () => {
    // Arrange
    router.navigate('/pages');

    // Act
    render(Header, { props: {} });

    // Assert
    expect(screen.getByRole('heading', { name: 'Pages' })).toBeTruthy();
  });

  it('shows the collection label for a content type path', () => {
    // Arrange
    schema.schema = { modules: {}, resources: [{ id: 'faq', label: 'FAQ', plural: 'FAQs', source: 'content_type' }] } as any;
    router.navigate('/content/faq');

    // Act
    render(Header, { props: {} });

    // Assert
    expect(screen.getByRole('heading', { name: 'FAQs' })).toBeTruthy();
  });

  it('toggles the desktop sidebar when the toggle button is clicked at desktop width', async () => {
    // Arrange
    window.innerWidth = 1280;
    render(Header, { props: {} });
    const before = sidebar.open;

    // Act
    await fireEvent.click(screen.getByTitle('Toggle Sidebar'));

    // Assert
    expect(sidebar.open).toBe(!before);
    expect(sidebar.mobileOpen).toBe(false);
  });

  it('toggles the mobile sidebar when the toggle button is clicked at phone width', async () => {
    // Arrange
    window.innerWidth = 400;
    render(Header, { props: {} });

    // Act
    await fireEvent.click(screen.getByTitle('Toggle Sidebar'));

    // Assert
    expect(sidebar.mobileOpen).toBe(true);
  });

  it('opens the search modal from the search button', async () => {
    // Arrange
    render(Header, { props: {} });

    // Act
    await fireEvent.click(screen.getByTitle('Quick search (⌘K or Ctrl+K)'));

    // Assert
    expect(screen.getByRole('dialog')).toBeTruthy();
  });

  it('toggles the theme when the theme button is clicked', async () => {
    // Arrange
    theme.setTheme('light');
    render(Header, { props: {} });

    // Act
    await fireEvent.click(screen.getByLabelText('Toggle theme'));

    // Assert
    expect(theme.isDark).toBe(true);
  });
});
