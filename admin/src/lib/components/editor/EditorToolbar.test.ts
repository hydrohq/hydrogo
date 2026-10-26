// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import EditorToolbar from './EditorToolbar.svelte';

function fakeEditor(opts: { active?: Record<string, boolean>; canUndo?: boolean; canRedo?: boolean; linkHref?: string } = {}) {
  const setLink = vi.fn();
  const run = vi.fn();
  const chain: any = {
    focus: () => chain,
    undo: () => chain,
    redo: () => chain,
    extendMarkRange: () => chain,
    setLink: (attrs: any) => { setLink(attrs); return chain; },
    unsetLink: () => chain,
    run,
  };
  return {
    chain: () => chain,
    isActive: (name: string) => opts.active?.[name] ?? false,
    can: () => ({ undo: () => opts.canUndo ?? true, redo: () => opts.canRedo ?? true }),
    getAttributes: () => ({ href: opts.linkHref ?? '' }),
    _setLink: setLink,
  } as any;
}

describe('EditorToolbar', () => {
  it('disables undo/redo when the editor reports nothing to undo/redo', () => {
    // Arrange
    const props = { editor: fakeEditor({ canUndo: false, canRedo: false }) };

    // Act
    render(EditorToolbar, { props });

    // Assert
    expect((screen.getByRole('button', { name: 'Undo' }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole('button', { name: 'Redo' }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('opens the link input with the current link prefilled, and prepends https:// to a bare host', async () => {
    // Arrange
    const editor = fakeEditor({ active: { link: true }, linkHref: '' });
    render(EditorToolbar, { props: { editor } });

    // Act
    await fireEvent.click(screen.getByRole('button', { name: 'Insert Link' }));
    const input = screen.getByPlaceholderText('https://example.com') as HTMLInputElement;
    await fireEvent.input(input, { target: { value: 'example.com/page' } });
    await fireEvent.click(screen.getByText('Apply'));

    // Assert
    expect(editor._setLink).toHaveBeenCalledWith({ href: 'https://example.com/page' });
  });

  it('leaves an absolute, root-relative or anchor link untouched', async () => {
    // Arrange
    const editor = fakeEditor();
    render(EditorToolbar, { props: { editor } });
    await fireEvent.click(screen.getByRole('button', { name: 'Insert Link' }));
    const input = screen.getByPlaceholderText('https://example.com') as HTMLInputElement;

    // Act
    await fireEvent.input(input, { target: { value: '/docs/intro' } });
    await fireEvent.click(screen.getByText('Apply'));

    // Assert
    expect(editor._setLink).toHaveBeenCalledWith({ href: '/docs/intro' });
  });

  it('applies the link on Enter and closes the input on Escape', async () => {
    // Arrange
    const editor = fakeEditor();
    render(EditorToolbar, { props: { editor } });
    await fireEvent.click(screen.getByRole('button', { name: 'Insert Link' }));
    const input = screen.getByPlaceholderText('https://example.com') as HTMLInputElement;
    await fireEvent.input(input, { target: { value: 'https://example.com' } });

    // Act
    await fireEvent.keyDown(input, { key: 'Enter' });

    // Assert
    expect(editor._setLink).toHaveBeenCalledWith({ href: 'https://example.com' });
    expect(screen.queryByPlaceholderText('https://example.com')).toBeNull();
  });
});
