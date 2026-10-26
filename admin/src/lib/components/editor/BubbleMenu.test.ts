// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import BubbleMenu from './BubbleMenu.svelte';

function fakeEditor(active: Record<string, boolean> = {}) {
  const run = vi.fn();
  const chain: any = {
    focus: () => chain,
    toggleBold: () => chain,
    toggleItalic: () => chain,
    toggleStrike: () => chain,
    toggleCode: () => chain,
    toggleHeading: () => chain,
    toggleBlockquote: () => chain,
    run,
  };
  return {
    chain: () => chain,
    isActive: (name: string) => active[name] ?? false,
    _run: run,
  } as any;
}

describe('BubbleMenu', () => {
  it('positions itself using the given top/left offsets', () => {
    // Arrange
    const props = { editor: fakeEditor(), position: { top: 50, left: 30 }, onOpenLink: vi.fn() };

    // Act
    const { container } = render(BubbleMenu, { props });

    // Assert
    const menu = container.firstElementChild as HTMLElement;
    expect(menu.style.top).toBe('50px');
    expect(menu.style.left).toBe('30px');
  });

  it('highlights the bold button when the active mark is bold', () => {
    // Arrange
    const props = { editor: fakeEditor({ bold: true }), position: { top: 0, left: 0 }, onOpenLink: vi.fn() };

    // Act
    render(BubbleMenu, { props });

    // Assert
    expect(screen.getByRole('button', { name: 'Bold' }).className).toContain('text-primary');
  });

  it('runs the bold command through the editor chain when clicked', async () => {
    // Arrange
    const editor = fakeEditor();
    const props = { editor, position: { top: 0, left: 0 }, onOpenLink: vi.fn() };
    render(BubbleMenu, { props });

    // Act
    await fireEvent.click(screen.getByRole('button', { name: 'Bold' }));

    // Assert
    expect(editor._run).toHaveBeenCalledOnce();
  });

  it('calls onOpenLink when the link button is clicked', async () => {
    // Arrange
    const onOpenLink = vi.fn();
    const props = { editor: fakeEditor(), position: { top: 0, left: 0 }, onOpenLink };
    render(BubbleMenu, { props });

    // Act
    await fireEvent.click(screen.getByRole('button', { name: 'Insert Link' }));

    // Assert
    expect(onOpenLink).toHaveBeenCalledOnce();
  });
});
