// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import PageSpinner from './PageSpinner.svelte';

describe('PageSpinner', () => {
  it('renders an accessible loading status', () => {
    render(PageSpinner, { props: {} });
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('can fill the whole screen', () => {
    render(PageSpinner, { props: { fullscreen: true } });
    expect(screen.getByTestId('page-spinner').className).toContain('min-h-screen');
  });
});
