// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ConfirmDialog from './ConfirmDialog.svelte';

describe('ConfirmDialog', () => {
  it('renders nothing when closed', () => {
    // Arrange
    const props = { open: false, title: 'Delete page', onconfirm: vi.fn(), oncancel: vi.fn() };

    // Act
    render(ConfirmDialog, { props });

    // Assert
    expect(screen.queryByRole('alertdialog')).toBeNull();
  });

  it('shows the title and calls oncancel when the cancel button is clicked', async () => {
    // Arrange
    const oncancel = vi.fn();
    const props = { open: true, title: 'Delete page', onconfirm: vi.fn(), oncancel };
    render(ConfirmDialog, { props });

    // Act
    expect(screen.getByText('Delete page')).toBeTruthy();
    await fireEvent.click(screen.getByText('Cancel'));

    // Assert
    expect(oncancel).toHaveBeenCalledOnce();
  });

  it('calls onconfirm when the confirm button is clicked', async () => {
    // Arrange
    const onconfirm = vi.fn();
    const props = { open: true, title: 'Delete page', onconfirm, oncancel: vi.fn() };
    render(ConfirmDialog, { props });

    // Act
    await fireEvent.click(screen.getByText('Delete'));

    // Assert
    expect(onconfirm).toHaveBeenCalledOnce();
  });

  it('disables both buttons while loading', () => {
    // Arrange
    const props = { open: true, title: 'Delete page', loading: true, onconfirm: vi.fn(), oncancel: vi.fn() };

    // Act
    render(ConfirmDialog, { props });

    // Assert
    expect((screen.getByText('Cancel') as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByText('Delete').closest('button') as HTMLButtonElement).disabled).toBe(true);
  });
});
