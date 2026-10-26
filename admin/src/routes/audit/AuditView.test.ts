// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { schema } from '$lib/stores/schema.svelte';
import { api } from '$lib/api/client';
import AuditView from './AuditView.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

function entry(over: Partial<any> = {}) {
  return { id: 'e1', at: '2024-01-01T00:00:00Z', actor: 'ada@example.com', action: 'content.update', resource: 'pages', target: 'home', ...over };
}

describe('AuditView', () => {
  afterEach(() => {
    schema.schema = null as any;
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
    vi.mocked(api.delete).mockReset();
  });

  it('lists loaded audit entries', async () => {
    // Arrange
    schema.schema = { modules: { site: false }, resources: [] } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.startsWith('/api/v1/admin/audit')) return Promise.resolve({ data: [entry()], metadata: { total: 1 } });
      return Promise.resolve({ data: [] });
    });

    // Act
    render(AuditView, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('ada@example.com')).toBeTruthy());
  });

  it('shows the empty state when there are no entries', async () => {
    // Arrange
    schema.schema = { modules: { site: false }, resources: [] } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [], metadata: { total: 0 } });

    // Act
    render(AuditView, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('No audit events found.')).toBeTruthy());
  });

  it('filters the security tab to auth/security entries only', async () => {
    // Arrange
    schema.schema = { modules: { site: false }, resources: [] } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.startsWith('/api/v1/admin/audit')) {
        return Promise.resolve({
          data: [entry(), entry({ id: 'e2', actor: 'unknown', action: 'auth.login_failed', resource: 'auth' })],
          metadata: { total: 2 },
        });
      }
      return Promise.resolve({ data: [] });
    });
    render(AuditView, { props: {} });
    await waitFor(() => expect(screen.getByText('ada@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText(/Security & Sign-ins/));

    // Assert
    expect(screen.getByText('Sign-in Failed')).toBeTruthy();
    expect(screen.queryByText('ada@example.com')).toBeNull();
  });

  it('skips the 404 probe fetch when the site module is off', async () => {
    // Arrange
    schema.schema = { modules: { site: false }, resources: [] } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [], metadata: { total: 0 } });

    // Act
    render(AuditView, { props: {} });
    await waitFor(() => expect(screen.getByText('No audit events found.')).toBeTruthy());

    // Assert
    expect(api.get).not.toHaveBeenCalledWith(expect.stringContaining('not-found'));
  });

  it('lists 404 probe hits and converts one into a redirect', async () => {
    // Arrange
    schema.schema = { modules: { site: true }, resources: [] } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.startsWith('/api/v1/admin/audit')) return Promise.resolve({ data: [], metadata: { total: 0 } });
      if (url.startsWith('/api/v1/admin/site/not-found')) {
        return Promise.resolve({ data: [{ path: '/old-page', hits: 5, first_seen: '2024-01-01T00:00:00Z', last_seen: '2024-01-02T00:00:00Z' }] });
      }
      return Promise.resolve({ data: [] });
    });
    vi.mocked(api.post).mockResolvedValue({});
    render(AuditView, { props: {} });
    await waitFor(() => expect(screen.getByText('No audit events found.')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText(/404 Scanner Log/));
    await waitFor(() => expect(screen.getByText('/old-page')).toBeTruthy());
    await fireEvent.click(screen.getByText('Redirect /'));

    // Assert
    await waitFor(() => expect(api.post).toHaveBeenCalledWith('/api/v1/admin/site/not-found/convert', { path: '/old-page', target: '/' }));
  });

  it('opens the inspector modal for an entry', async () => {
    // Arrange
    schema.schema = { modules: { site: false }, resources: [] } as any;
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.startsWith('/api/v1/admin/audit')) return Promise.resolve({ data: [entry()], metadata: { total: 1 } });
      return Promise.resolve({ data: [] });
    });
    render(AuditView, { props: {} });
    await waitFor(() => expect(screen.getByText('ada@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Inspect'));

    // Assert
    expect(screen.getByRole('dialog')).toBeTruthy();
  });
});
