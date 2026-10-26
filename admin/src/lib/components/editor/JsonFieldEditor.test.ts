// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import JsonFieldEditor from './JsonFieldEditor.svelte';

describe('JsonFieldEditor', () => {
  it('starts in form mode with an empty object showing "no fields" and lets a field be added', async () => {
    // Arrange
    const props = { value: {} };

    // Act
    render(JsonFieldEditor, { props });

    // Assert
    expect(screen.getByText('No key/value fields defined yet.')).toBeTruthy();
    await fireEvent.click(screen.getByText('Add field / option'));
    expect(screen.queryByText('No key/value fields defined yet.')).toBeNull();
  });

  it('renders an existing object as key/value rows', () => {
    // Arrange
    const props = { value: { color: 'blue', size: 10 } };

    // Act
    render(JsonFieldEditor, { props });

    // Assert
    expect(screen.getByDisplayValue('color')).toBeTruthy();
    expect(screen.getByDisplayValue('blue')).toBeTruthy();
    expect(screen.getByDisplayValue('size')).toBeTruthy();
    expect(screen.getByDisplayValue('10')).toBeTruthy();
  });

  it('renders an existing array as indexed item rows', () => {
    // Arrange
    const props = { value: ['a', 'b'] };

    // Act
    render(JsonFieldEditor, { props });

    // Assert
    expect(screen.getByText('#1')).toBeTruthy();
    expect(screen.getByText('#2')).toBeTruthy();
    expect(screen.getByDisplayValue('a')).toBeTruthy();
    expect(screen.getByDisplayValue('b')).toBeTruthy();
  });

  it('switches to raw mode and shows a parse error for invalid JSON', async () => {
    // Arrange
    const props = { value: { a: 1 } };
    render(JsonFieldEditor, { props });

    // Act
    await fireEvent.click(screen.getByText('Raw JSON'));
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement;
    await fireEvent.input(textarea, { target: { value: '{ not json' } });

    // Assert
    expect(screen.getByText(/^Error:/)).toBeTruthy();
  });

  it('starts in raw mode for a string value that is not valid JSON', () => {
    // Arrange
    const props = { value: 'not json at all' };

    // Act
    render(JsonFieldEditor, { props });

    // Assert
    expect(screen.getByDisplayValue('not json at all')).toBeTruthy();
    expect(screen.getByText('Error: Not valid JSON')).toBeTruthy();
  });
});
