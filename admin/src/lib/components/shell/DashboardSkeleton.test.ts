// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import DashboardSkeleton from './DashboardSkeleton.svelte';

describe('DashboardSkeleton (Adaptive Content Skeleton)', () => {
  it('renders Console skeleton with Mağaza Fabrikası title and store table when on /console', () => {
    render(DashboardSkeleton, { props: { path: '/console' } });
    expect(screen.getAllByText('Mağaza Fabrikası').length).toBeGreaterThan(0);
    expect(screen.getByText('Console')).toBeTruthy();
    expect(screen.getByText('Aktif Mağaza')).toBeTruthy();
    expect(screen.getByText('Kalan Kredi')).toBeTruthy();
    expect(screen.getByText('RAM / Sistem')).toBeTruthy();
    expect(screen.getByText('DNS / SSL')).toBeTruthy();
    expect(screen.getByText('Özel alan adı')).toBeTruthy();
  });

  it('renders Table skeleton when on /pages', () => {
    render(DashboardSkeleton, { props: { path: '/pages' } });
    expect(screen.getByRole('heading', { name: 'Pages' })).toBeTruthy();
  });

  it('renders Editor skeleton when on /pages/new', () => {
    const { container } = render(DashboardSkeleton, { props: { path: '/pages/new' } });
    expect(container.querySelector('.min-h-\\[300px\\]')).toBeTruthy();
  });

  it('renders minimal Auth card skeleton without sidebar when on /login', () => {
    render(DashboardSkeleton, { props: { path: '/login' } });
    expect(screen.queryByText('Starting HydroGo Admin…')).toBeNull();
  });
});
