// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import MembersView from './MembersView.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), patch: vi.fn(), post: vi.fn() },
}));

function member(over: Partial<any> = {}) {
  return { id: 'm1', email: 'carol@example.com', active: true, created_at: '2024-01-01T00:00:00Z', ...over };
}

describe('MembersView', () => {
  afterEach(() => {
    vi.mocked(api.get).mockReset();
    vi.mocked(api.patch).mockReset();
    vi.mocked(api.post).mockReset();
  });

  it('lists loaded members', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [member()] });

    // Act
    render(MembersView, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('carol@example.com')).toBeTruthy());
  });

  it('shows the empty state when there are no members', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [] });

    // Act
    render(MembersView, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('No registered members found.')).toBeTruthy());
  });

  it('filters members by the search box', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [member(), member({ id: 'm2', email: 'dave@example.com' })] });
    render(MembersView, { props: {} });
    await waitFor(() => expect(screen.getByText('carol@example.com')).toBeTruthy());

    // Act
    await fireEvent.input(screen.getByPlaceholderText('Search members...'), { target: { value: 'dave' } });

    // Assert
    expect(screen.getByText('dave@example.com')).toBeTruthy();
    expect(screen.queryByText('carol@example.com')).toBeNull();
  });

  it('filters members to only inactive ones', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({
      data: [member(), member({ id: 'm2', email: 'dave@example.com', active: false })],
    });
    render(MembersView, { props: {} });
    await waitFor(() => expect(screen.getByText('carol@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Inactive'));

    // Assert
    expect(screen.queryByText('carol@example.com')).toBeNull();
    expect(screen.getByText('dave@example.com')).toBeTruthy();
  });

  it('toggles a member active state from the row button', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [member()] });
    vi.mocked(api.patch).mockResolvedValue({});
    render(MembersView, { props: {} });
    await waitFor(() => expect(screen.getByText('carol@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('Disable'));

    // Assert
    expect(api.patch).toHaveBeenCalledWith('/api/v1/admin/members/m1', { active: false });
    await waitFor(() => expect(screen.getByText('Enable')).toBeTruthy());
  });

  it('opens the detail modal and marks the email verified', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [member()] });
    vi.mocked(api.post).mockResolvedValue({});
    render(MembersView, { props: {} });
    await waitFor(() => expect(screen.getByText('carol@example.com')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('carol@example.com'));
    await fireEvent.click(screen.getByText('Mark Email as Verified'));

    // Assert
    await waitFor(() => expect(screen.getByText('Email verified successfully.')).toBeTruthy());
    expect(api.post).toHaveBeenCalledWith('/api/v1/admin/members/m1/verify-email');
  });

  it('revokes all sessions for a member through the confirm dialog', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [member()] });
    vi.mocked(api.post).mockResolvedValue({});
    render(MembersView, { props: {} });
    await waitFor(() => expect(screen.getByText('carol@example.com')).toBeTruthy());
    await fireEvent.click(screen.getByText('carol@example.com'));

    // Act
    await fireEvent.click(screen.getByText('Revoke All Active Sessions'));
    await fireEvent.click(screen.getByText('Revoke Sessions'));

    // Assert
    await waitFor(() => expect(api.post).toHaveBeenCalledWith('/api/v1/admin/members/m1/sessions/revoke-all'));
  });
});
