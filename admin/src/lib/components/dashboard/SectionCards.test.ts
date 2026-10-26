// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import SectionCards from './SectionCards.svelte';

describe('SectionCards', () => {
  it('shows zero counts and "System Online" when no metrics are given', () => {
    // Arrange & Act
    render(SectionCards, { props: {} });

    // Assert
    expect(screen.getByText('System Online')).toBeTruthy();
    expect(screen.getByText('< 5ms')).toBeTruthy();
  });

  it('formats members, requests and latency from the given metrics', () => {
    // Arrange
    const props = { metrics: { totalMembers: 12345, totalRequests: 98765, avgLatencyMs: 42 } };

    // Act
    render(SectionCards, { props });

    // Assert
    expect(screen.getByText('12,345')).toBeTruthy();
    expect(screen.getByText('98,765')).toBeTruthy();
    expect(screen.getByText('42ms')).toBeTruthy();
  });

  it('formats uptime in minutes, hours and days depending on magnitude', () => {
    // Arrange, Act & Assert: minutes
    render(SectionCards, { props: { metrics: { uptimeSeconds: 90 } } });
    expect(screen.getByText('1m uptime')).toBeTruthy();
  });

  it('formats uptime in hours once past 60 minutes', () => {
    // Arrange & Act
    render(SectionCards, { props: { metrics: { uptimeSeconds: 2 * 3600 + 5 * 60 } } });

    // Assert
    expect(screen.getByText('2h 5m uptime')).toBeTruthy();
  });

  it('formats uptime in days once past 24 hours', () => {
    // Arrange & Act
    render(SectionCards, { props: { metrics: { uptimeSeconds: 25 * 3600 } } });

    // Assert
    expect(screen.getByText('1d 1h uptime')).toBeTruthy();
  });
});
