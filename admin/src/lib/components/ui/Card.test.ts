// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { createRawSnippet } from 'svelte';
import { render, screen } from '@testing-library/svelte';
import Card from './Card.svelte';

function textSnippet(text: string) {
  return createRawSnippet(() => ({ render: () => `<p>${text}</p>` }));
}

describe('Card', () => {
  it('renders title and description', () => {
    // Arrange
    const props = { title: 'Storage', description: 'Media usage this month' };

    // Act
    render(Card, { props });

    // Assert
    expect(screen.getByText('Storage')).toBeTruthy();
    expect(screen.getByText('Media usage this month')).toBeTruthy();
  });

  it('omits the header entirely when there is no title, description or headerAction', () => {
    // Arrange
    const props = { children: textSnippet('Body content') };

    // Act
    const { container } = render(Card, { props });

    // Assert
    expect(container.querySelector('.border-border\\/60')).toBeNull();
    expect(screen.getByText('Body content')).toBeTruthy();
  });

  it('renders the footer snippet when given', () => {
    // Arrange
    const props = { footer: textSnippet('Last synced 2 minutes ago') };

    // Act
    render(Card, { props });

    // Assert
    expect(screen.getByText('Last synced 2 minutes ago')).toBeTruthy();
  });
});
