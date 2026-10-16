import { describe, it, expect } from 'vitest';
import { router } from './router.svelte';

describe('Router', () => {
  it('parses empty and default hash correctly', () => {
    router.hash = '#/';
    expect(router.path).toBe('/');
    expect(router.query.toString()).toBe('');
  });

  it('parses nested paths correctly', () => {
    router.hash = '#/content/faq/new';
    expect(router.path).toBe('/content/faq/new');
  });

  it('parses query parameters correctly', () => {
    router.hash = '#/pages?status=draft&locale=en';
    expect(router.path).toBe('/pages');
    expect(router.query.get('status')).toBe('draft');
    expect(router.query.get('locale')).toBe('en');
  });

  it('navigates with clean paths without hash', () => {
    router.navigate('/system');
    expect(router.path).toBe('/system');

    router.navigate('/content/faq');
    expect(router.path).toBe('/content/faq');

    router.navigate('/admin/dashboard');
    expect(router.path).toBe('/dashboard');
  });

  it('handles query parameters in clean path navigation', () => {
    router.navigate('/news?status=published&page=2');
    expect(router.path).toBe('/news');
    expect(router.query.get('status')).toBe('published');
    expect(router.query.get('page')).toBe('2');
  });

  it('protects navigation when dirty and allows cancel or confirm', () => {
    router.navigate('/dashboard');
    router.setDirty(true, 'Unsaved changes');
    expect(router.isDirty).toBe(true);

    // Navigation attempt should be intercepted
    router.navigate('/news');
    expect(router.showUnsavedModal).toBe(true);
    expect(router.pendingTarget).toBe('/news');
    expect(router.path).toBe('/dashboard'); // Path has not changed

    // Cancel leave
    router.cancelLeave();
    expect(router.showUnsavedModal).toBe(false);
    expect(router.pendingTarget).toBe(null);
    expect(router.path).toBe('/dashboard');

    // Attempt again and confirm leave
    router.navigate('/news');
    expect(router.showUnsavedModal).toBe(true);
    router.confirmLeave();
    expect(router.showUnsavedModal).toBe(false);
    expect(router.isDirty).toBe(false);
    expect(router.path).toBe('/news');
  });
});
