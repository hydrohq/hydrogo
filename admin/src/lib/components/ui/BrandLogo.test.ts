// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import { brandingStore } from '$lib/stores/branding.svelte';
import BrandLogo from './BrandLogo.svelte';

describe('BrandLogo', () => {
  afterEach(() => {
    brandingStore.set({ name: '', logo_url: undefined, color: undefined } as any);
  });

  it('falls back to the HydroGo name when no branding has loaded', () => {
    // Arrange
    brandingStore.set({} as any);

    // Act
    render(BrandLogo, { props: {} });

    // Assert
    expect(screen.getByText('HydroGo')).toBeTruthy();
  });

  it('shows the project name once branding loads', () => {
    // Arrange
    brandingStore.set({ name: 'Acme Admin' } as any);

    // Act
    render(BrandLogo, { props: {} });

    // Assert
    expect(screen.getByText('Acme Admin')).toBeTruthy();
  });

  it('renders a logo image instead of the icon when a logo_url is set', () => {
    // Arrange
    brandingStore.set({ name: 'Acme', logo_url: 'https://example.com/logo.png' } as any);

    // Act
    const { container } = render(BrandLogo, { props: {} });

    // Assert
    const img = container.querySelector('img');
    expect(img?.getAttribute('src')).toBe('https://example.com/logo.png');
  });

  it('hides the text when showText is false', () => {
    // Arrange
    brandingStore.set({ name: 'Acme' } as any);

    // Act
    render(BrandLogo, { props: { showText: false } });

    // Assert
    expect(screen.queryByText('Acme')).toBeNull();
  });
});
