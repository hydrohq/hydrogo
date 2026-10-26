// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import type { PresenceState } from '$lib/presence.svelte';
import type { ActiveEditor } from '$lib/types/presence';
import DocumentPresence from './DocumentPresence.svelte';

function fakePresence(editors: ActiveEditor[], dismissedBanner = false): PresenceState {
  return {
    activeEditors: editors,
    count: editors.length,
    dismissedBanner,
    names: editors.map((e) => e.name).join(', '),
    dismiss: vi.fn(),
  } as any;
}

const editor = (over: Partial<ActiveEditor> = {}): ActiveEditor => ({
  name: 'Ada Lovelace',
  email: 'ada@example.com',
  updated_at: new Date().toISOString(),
  active_seconds: 5,
  ...over,
});

describe('DocumentPresence', () => {
  it('renders nothing when no one else is editing', () => {
    // Arrange
    const props = { presence: fakePresence([]), endpoint: '/x', resourceId: '1' };

    // Act
    const { container } = render(DocumentPresence, { props });

    // Assert
    expect(container.textContent?.trim()).toBe('');
  });

  it('shows the single editor by name in the pill', () => {
    // Arrange
    const props = { presence: fakePresence([editor()]), endpoint: '/x', resourceId: '1' };

    // Act
    render(DocumentPresence, { props });

    // Assert
    expect(screen.getByText('Ada Lovelace')).toBeTruthy();
  });

  it('shows a count for multiple editors instead of a name', () => {
    // Arrange
    const props = { presence: fakePresence([editor(), editor({ name: 'Grace Hopper', email: 'grace@example.com' })]), endpoint: '/x', resourceId: '1' };

    // Act
    render(DocumentPresence, { props });

    // Assert
    expect(screen.getByText('2 people')).toBeTruthy();
  });

  it('opens the popover with each editor listed, and toggles closed again', async () => {
    // Arrange
    const props = { presence: fakePresence([editor()]), endpoint: '/x', resourceId: '1' };
    render(DocumentPresence, { props });

    // Act
    await fireEvent.click(screen.getByTitle('This document is currently being viewed/edited by someone else too'));

    // Assert
    expect(screen.getByText('ada@example.com')).toBeTruthy();
  });

  it('calls dismiss on the presence state when the warning banner is closed', async () => {
    // Arrange
    const presence = fakePresence([editor()]);
    const props = { presence, endpoint: '/x', resourceId: '1', mode: 'banner' as const };
    render(DocumentPresence, { props });

    // Act
    await fireEvent.click(screen.getByTitle('Dismiss warning'));

    // Assert
    expect(presence.dismiss).toHaveBeenCalledOnce();
  });

  it('hides the banner once dismissedBanner is true', () => {
    // Arrange
    const props = { presence: fakePresence([editor()], true), endpoint: '/x', resourceId: '1', mode: 'banner' as const };

    // Act
    render(DocumentPresence, { props });

    // Assert
    expect(screen.queryByTitle('Dismiss')).toBeNull();
  });
});
