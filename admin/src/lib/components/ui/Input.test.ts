// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Input from './Input.svelte';

describe('Input', () => {
  it('renders a label wired to the input via a generated id', () => {
    // Arrange
    const props = { label: 'Title' };

    // Act
    render(Input, { props });

    // Assert
    const input = screen.getByLabelText('Title') as HTMLInputElement;
    expect(input).toBeTruthy();
  });

  it('uses the given id instead of generating one', () => {
    // Arrange
    const props = { label: 'Title', id: 'page-title' };

    // Act
    render(Input, { props });

    // Assert
    expect(screen.getByLabelText('Title').id).toBe('page-title');
  });

  it('shows the error message instead of the hint when both are given', () => {
    // Arrange
    const props = { label: 'Title', error: 'Required', hint: 'Shown on the page' };

    // Act
    render(Input, { props });

    // Assert
    expect(screen.getByText('Required')).toBeTruthy();
    expect(screen.queryByText('Shown on the page')).toBeNull();
  });

  it('updates the bound value as the user types', async () => {
    // Arrange
    const props = { label: 'Title', value: '' };

    // Act
    render(Input, { props });
    const input = screen.getByLabelText('Title') as HTMLInputElement;
    await fireEvent.input(input, { target: { value: 'Hello' } });

    // Assert
    expect(input.value).toBe('Hello');
  });
});
