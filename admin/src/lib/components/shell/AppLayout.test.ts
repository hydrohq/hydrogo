// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { createRawSnippet } from 'svelte';
import { render, screen } from '@testing-library/svelte';
import { schema } from '$lib/stores/schema.svelte';
import AppLayout from './AppLayout.svelte';

describe('AppLayout', () => {
  it('renders the sidebar, header and the given children', () => {
    // Arrange
    schema.schema = null as any;
    const children = createRawSnippet(() => ({ render: () => '<p>Page body</p>' }));

    // Act
    render(AppLayout, { props: { children } });

    // Assert
    expect(screen.getByText('Page body')).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeTruthy();
  });
});
