import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';

afterEach(() => {
  cleanup();
});

// jsdom has no matchMedia; the theme store calls it unconditionally when
// window exists, so component tests that import it (directly or via
// Header/Sidebar) need this stub.
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }) as unknown as MediaQueryList;
}
