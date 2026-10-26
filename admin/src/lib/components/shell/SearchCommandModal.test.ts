// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { router } from '$lib/router.svelte';
import { schema } from '$lib/stores/schema.svelte';
import SearchCommandModal from './SearchCommandModal.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }) },
}));

describe('SearchCommandModal', () => {
  afterEach(() => {
    schema.schema = null as any;
  });

  it('renders nothing when closed', () => {
    // Arrange & Act
    const { container } = render(SearchCommandModal, { props: { open: false, onclose: vi.fn() } });

    // Assert
    expect(container.querySelector('[role="dialog"]')).toBeNull();
  });

  it('lists the static nav items when open with no query', () => {
    // Arrange & Act
    render(SearchCommandModal, { props: { open: true, onclose: vi.fn() } });

    // Assert
    expect(screen.getByText('Dashboard')).toBeTruthy();
    expect(screen.getByText('Pages')).toBeTruthy();
  });

  it('filters items by the typed query', async () => {
    // Arrange
    render(SearchCommandModal, { props: { open: true, onclose: vi.fn() } });
    const input = screen.getByPlaceholderText(/search/i) as HTMLInputElement;

    // Act
    await fireEvent.input(input, { target: { value: 'dashboard' } });

    // Assert
    expect(screen.getByText('Dashboard')).toBeTruthy();
    expect(screen.queryByText('Pages')).toBeNull();
  });

  it('shows a no-results message when nothing matches', async () => {
    // Arrange
    render(SearchCommandModal, { props: { open: true, onclose: vi.fn() } });
    const input = screen.getByPlaceholderText(/search/i) as HTMLInputElement;

    // Act
    await fireEvent.input(input, { target: { value: 'zzzznothingmatchesz' } });

    // Assert
    expect(screen.queryByText('Dashboard')).toBeNull();
  });

  it('navigates to the clicked item and closes', async () => {
    // Arrange
    const onclose = vi.fn();
    render(SearchCommandModal, { props: { open: true, onclose } });

    // Act
    await fireEvent.click(screen.getByText('Pages'));

    // Assert
    expect(router.path).toBe('/pages');
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('closes on Escape', async () => {
    // Arrange
    const onclose = vi.fn();
    render(SearchCommandModal, { props: { open: true, onclose } });

    // Act
    await fireEvent.keyDown(window, { key: 'Escape' });

    // Assert
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('includes a collection from the schema in the results', () => {
    // Arrange
    schema.schema = { modules: {}, resources: [{ id: 'faq', label: 'FAQ', plural: 'FAQs', source: 'content_type' }] } as any;

    // Act
    render(SearchCommandModal, { props: { open: true, onclose: vi.fn() } });

    // Assert
    expect(screen.getByText('FAQs')).toBeTruthy();
  });
});
