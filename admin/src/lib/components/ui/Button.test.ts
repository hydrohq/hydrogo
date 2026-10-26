// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { createRawSnippet } from 'svelte';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Button from './Button.svelte';

function textSnippet(text: string) {
  return createRawSnippet(() => ({ render: () => `<span>${text}</span>` }));
}

describe('Button', () => {
  it('renders its label and responds to clicks', async () => {
    // Arrange
    const onclick = vi.fn();
    const props = { children: textSnippet('Save'), onclick };

    // Act
    render(Button, { props });
    await fireEvent.click(screen.getByText('Save'));

    // Assert
    expect(onclick).toHaveBeenCalledOnce();
  });

  it('disables the button and ignores clicks while loading', async () => {
    // Arrange
    const onclick = vi.fn();
    const props = { children: textSnippet('Save'), onclick, loading: true };

    // Act
    render(Button, { props });
    const button = screen.getByRole('button') as HTMLButtonElement;

    // Assert
    expect(button.disabled).toBe(true);
  });

  it('is disabled when the disabled prop is set, independent of loading', () => {
    // Arrange
    const props = { children: textSnippet('Save'), disabled: true };

    // Act
    render(Button, { props });

    // Assert
    expect((screen.getByRole('button') as HTMLButtonElement).disabled).toBe(true);
  });
});
