// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { schema } from '$lib/stores/schema.svelte';
import { api } from '$lib/api/client';
import ChartAreaInteractive from './ChartAreaInteractive.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn() },
}));

describe('ChartAreaInteractive', () => {
  afterEach(() => {
    schema.schema = null as any;
    vi.mocked(api.get).mockReset();
  });

  it('shows the loading state, then "no categories" once the news module returns none', async () => {
    // Arrange
    schema.schema = { modules: { news: true } } as any;
    vi.mocked(api.get).mockResolvedValue({ data: [] });

    // Act
    render(ChartAreaInteractive, { props: {} });

    // Assert
    expect(screen.getByText('Loading category data...')).toBeTruthy();
    await waitFor(() => expect(screen.getByText('No category data found yet.')).toBeTruthy());
  });

  it('skips the fetch and shows "no categories" when the news module is off', async () => {
    // Arrange
    schema.schema = { modules: { news: false } } as any;

    // Act
    render(ChartAreaInteractive, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('No category data found yet.')).toBeTruthy());
    expect(api.get).not.toHaveBeenCalled();
  });

  it('groups loaded articles by category and shows the article count', async () => {
    // Arrange
    schema.schema = { modules: { news: true } } as any;
    vi.mocked(api.get).mockResolvedValue({
      data: [{ category: 'Tech' }, { category: 'Tech' }, { category: 'Sports' }],
    });

    // Act
    render(ChartAreaInteractive, { props: {} });

    // Assert
    await waitFor(() => expect(screen.getByText('Tech')).toBeTruthy());
    expect(screen.getByText('Sports')).toBeTruthy();
    expect(screen.getByText('3 articles')).toBeTruthy();
  });

  it('switches to the traffic view and totals desktop/mobile visits from the analytics prop', async () => {
    // Arrange
    schema.schema = { modules: { news: false } } as any;
    const today = new Date().toISOString().slice(0, 10);
    const props = {
      analytics: {
        daily: [
          { Day: today, Metric: 'device', Dimension: 'desktop', EventCount: 7 },
          { Day: today, Metric: 'device', Dimension: 'mobile', EventCount: 3 },
        ],
      },
    };
    render(ChartAreaInteractive, { props });

    // Act
    await fireEvent.click(screen.getByText('Visitor Chart'));

    // Assert
    expect(screen.getByText('Desktop: 7')).toBeTruthy();
    expect(screen.getByText('Mobile: 3')).toBeTruthy();
  });

  it('shows the "no traffic" footer note when there are zero visits', async () => {
    // Arrange
    schema.schema = { modules: { news: false } } as any;
    render(ChartAreaInteractive, { props: {} });

    // Act
    await fireEvent.click(screen.getByText('Visitor Chart'));

    // Assert
    expect(screen.getByText(/0 visits recorded in this period/)).toBeTruthy();
  });
});
