// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { createRawSnippet } from 'svelte';
import { render, screen } from '@testing-library/svelte';
import Badge from './Badge.svelte';

function textSnippet(text: string) {
  return createRawSnippet(() => ({
    render: () => `<span>${text}</span>`,
  }));
}

describe('Badge', () => {
  it('renders its children text', () => {
    // Arrange
    const props = { variant: 'success' as const, children: textSnippet('Published') };

    // Act
    render(Badge, { props });

    // Assert
    expect(screen.getByText('Published')).toBeTruthy();
  });

  it('falls back to the default variant style when given an unknown one', () => {
    // Arrange
    const props = { variant: 'not-a-real-variant' as any, children: textSnippet('Mystery') };

    // Act
    render(Badge, { props });

    // Assert
    const el = screen.getByText('Mystery').closest('span.inline-flex') as HTMLElement;
    expect(el.className).toContain('bg-primary');
  });
});
