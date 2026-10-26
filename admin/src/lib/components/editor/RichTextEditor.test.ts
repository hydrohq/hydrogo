// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/svelte';
import RichTextEditor from './RichTextEditor.svelte';

describe('RichTextEditor', () => {
  it('mounts a real TipTap editor with the toolbar and an empty word count', async () => {
    // Arrange & Act
    render(RichTextEditor, { props: { value: '' } });

    // Assert
    await waitFor(() => expect(screen.getByLabelText('Undo')).toBeTruthy());
    expect(screen.getByText('0 words')).toBeTruthy();
    expect(document.querySelector('.tiptap')).toBeTruthy();
  });

  it('loads the given HTML value into the editor content', async () => {
    // Arrange & Act
    render(RichTextEditor, { props: { value: '<p>Hello world</p>' } });

    // Assert
    await waitFor(() => expect(document.querySelector('.tiptap')?.textContent).toContain('Hello world'));
    expect(screen.getByText('2 words')).toBeTruthy();
  });

  it('shows the label and the error message when given', async () => {
    // Arrange & Act
    render(RichTextEditor, { props: { value: '', label: 'Body', error: 'Body is required' } });

    // Assert
    expect(screen.getByText('Body')).toBeTruthy();
    expect(screen.getByText('Body is required')).toBeTruthy();
  });
});
