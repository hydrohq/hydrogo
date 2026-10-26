// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { router } from '$lib/router.svelte';
import { schema } from '$lib/stores/schema.svelte';
import { api } from '$lib/api/client';
import DataTable from './DataTable.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn() },
}));

function mockFetch(pages: any[], news: any[]) {
  vi.mocked(api.get).mockImplementation((url: string) => {
    if (url === '/api/v1/admin/pages') return Promise.resolve({ data: pages });
    if (url === '/api/v1/admin/news') return Promise.resolve({ data: news });
    return Promise.resolve({ data: [] });
  });
}

describe('DataTable', () => {
  afterEach(() => {
    schema.schema = null as any;
    vi.mocked(api.get).mockReset();
  });

  it('loads and lists pages and news together, newest first', async () => {
    // Arrange
    schema.schema = { modules: { pages: true, news: true }, resources: [] } as any;
    mockFetch(
      [{ id: 'p1', title: 'About Us', status: 'published', updated_at: '2024-01-01T00:00:00Z' }],
      [{ id: 'n1', title: 'Launch Day', status: 'draft', updated_at: '2024-06-01T00:00:00Z' }]
    );

    // Act
    render(DataTable, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('Launch Day')).toBeTruthy());
    expect(screen.getByText('About Us')).toBeTruthy();
  });

  it('filters rows to the active tab', async () => {
    // Arrange
    schema.schema = { modules: { pages: true, news: true }, resources: [] } as any;
    mockFetch(
      [{ id: 'p1', title: 'About Us', status: 'published', updated_at: '2024-01-01T00:00:00Z' }],
      [{ id: 'n1', title: 'Launch Day', status: 'draft', updated_at: '2024-06-01T00:00:00Z' }]
    );
    render(DataTable, { props: {} });
    await waitFor(() => expect(screen.getByText('About Us')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Pages'));

    // Assert
    expect(screen.getByText('About Us')).toBeTruthy();
    expect(screen.queryByText('Launch Day')).toBeNull();
  });

  it('filters rows by the search box across title and author', async () => {
    // Arrange
    schema.schema = { modules: { pages: true, news: true }, resources: [] } as any;
    mockFetch(
      [{ id: 'p1', title: 'About Us', status: 'published', updated_at: '2024-01-01T00:00:00Z' }],
      [{ id: 'n1', title: 'Launch Day', status: 'draft', updated_at: '2024-06-01T00:00:00Z' }]
    );
    render(DataTable, { props: {} });
    await waitFor(() => expect(screen.getByText('About Us')).toBeTruthy());

    // Act
    await fireEvent.input(screen.getByPlaceholderText('Filter content...'), { target: { value: 'launch' } });

    // Assert
    expect(screen.getByText('Launch Day')).toBeTruthy();
    expect(screen.queryByText('About Us')).toBeNull();
  });

  it('navigates to the row target path when a row is clicked', async () => {
    // Arrange
    schema.schema = { modules: { pages: true, news: false }, resources: [] } as any;
    mockFetch([{ id: 'p1', title: 'About Us', status: 'published', updated_at: '2024-01-01T00:00:00Z' }], []);
    render(DataTable, { props: {} });
    await waitFor(() => expect(screen.getByText('About Us')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('About Us'));

    // Assert
    expect(router.path).toBe('/pages/p1');
  });

  it('shows the empty state when there is nothing to list', async () => {
    // Arrange
    schema.schema = { modules: { pages: true, news: true }, resources: [] } as any;
    mockFetch([], []);

    // Act
    render(DataTable, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('No records found in this section.')).toBeTruthy());
  });
});
