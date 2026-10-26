// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Modal from './Modal.svelte';

describe('Modal', () => {
  it('renders nothing when closed', () => {
    // Arrange
    const props = { open: false, title: 'Edit page' };

    // Act
    render(Modal, { props });

    // Assert
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('shows the title and description when open', () => {
    // Arrange
    const props = { open: true, title: 'Edit page', description: 'Change the page content' };

    // Act
    render(Modal, { props });

    // Assert
    expect(screen.getByText('Edit page')).toBeTruthy();
    expect(screen.getByText('Change the page content')).toBeTruthy();
  });

  it('calls onclose when the backdrop is clicked', async () => {
    // Arrange
    const onclose = vi.fn();
    const props = { open: true, title: 'Edit page', onclose };
    render(Modal, { props });

    // Act
    await fireEvent.click(screen.getByLabelText('Close modal backdrop'));

    // Assert
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('falls back to onClose when onclose is not given', async () => {
    // Arrange
    const onClose = vi.fn();
    const props = { open: true, title: 'Edit page', onClose };
    render(Modal, { props });

    // Act
    await fireEvent.click(screen.getByLabelText('Close modal backdrop'));

    // Assert
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('closes on Escape while open', async () => {
    // Arrange
    const onclose = vi.fn();
    const props = { open: true, title: 'Edit page', onclose };
    render(Modal, { props });

    // Act
    await fireEvent.keyDown(window, { key: 'Escape' });

    // Assert
    expect(onclose).toHaveBeenCalledOnce();
  });
});
