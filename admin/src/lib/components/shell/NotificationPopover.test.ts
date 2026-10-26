// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { schema } from '$lib/stores/schema.svelte';
import { api } from '$lib/api/client';
import NotificationPopover from './NotificationPopover.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

function notification(over: Partial<any> = {}) {
  return {
    id: 'n1',
    title: 'Comment added',
    message: 'Someone commented on your page',
    created_at: new Date().toISOString(),
    read: false,
    ...over,
  };
}

describe('NotificationPopover', () => {
  afterEach(() => {
    schema.schema = null as any;
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
  });

  it('does not fetch notifications when the pages module is off', async () => {
    // Arrange
    schema.schema = { modules: { pages: false } } as any;

    // Act
    render(NotificationPopover, { props: {} });
    await fireEvent.click(screen.getByLabelText('Notifications'));

    // Assert
    expect(api.get).not.toHaveBeenCalled();
  });

  it('loads and shows notifications with an unread badge once opened', async () => {
    // Arrange
    schema.schema = { modules: { pages: true } } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [notification()], metadata: { unread: 1 } });

    // Act
    render(NotificationPopover, { props: {} });
    await fireEvent.click(screen.getByLabelText('Notifications'));

    // Assert
    await waitFor(() => expect(screen.getByText('Comment added')).toBeTruthy());
  });

  it('shows an empty state when there are no notifications', async () => {
    // Arrange
    schema.schema = { modules: { pages: true } } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });

    // Act
    render(NotificationPopover, { props: {} });
    await fireEvent.click(screen.getByLabelText('Notifications'));

    // Assert
    await waitFor(() => expect(screen.getByText('No new notifications')).toBeTruthy());
  });

  it('marks all as read, clearing the unread badge', async () => {
    // Arrange
    schema.schema = { modules: { pages: true } } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [notification()], metadata: { unread: 1 } });
    vi.mocked(api.post).mockResolvedValue({});
    render(NotificationPopover, { props: {} });
    await fireEvent.click(screen.getByLabelText('Notifications'));
    await waitFor(() => expect(screen.getByText('Mark all read')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Mark all read'));

    // Assert
    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/notifications/read-all', {});
    await waitFor(() => expect(screen.queryByText('Mark all read')).toBeNull());
  });
});
