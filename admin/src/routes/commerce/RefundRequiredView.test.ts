// @vitest-environment jsdom
import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock('$lib/api/client', () => ({ api: { get, post }, ApiClientError: class ApiClientError extends Error {} }));
const { confirmAction } = vi.hoisted(() => ({ confirmAction: vi.fn() }));
vi.mock('$lib/confirm', () => ({ confirmAction }));
vi.mock('$lib/stores/auth.svelte', () => ({ auth: { hasPermission: () => true } }));

import RefundRequiredView from './RefundRequiredView.svelte';

describe('RefundRequiredView', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: [{ id: 'rr-1', order_id: 'order-1', provider: 'stripe', amount_cents: 1250, currency: 'USD', reason: 'Late payment', created_at: '2026-01-01T00:00:00Z' }] });
    post.mockReset().mockResolvedValue({ data: { resolved: true } });
    confirmAction.mockReset().mockResolvedValue(true);
  });

  it('lists refund requirements and resolves one with the operator note', async () => {
    render(RefundRequiredView);
    expect(await screen.findByText('Late payment')).toBeTruthy();
    const note = screen.getByLabelText('order-1 note');
    await fireEvent.input(note, { target: { value: 'Refund sent' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Resolve' }));
    expect(confirmAction).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith('/api/v1/admin/commerce/refunds-required/rr-1/resolve', { note: 'Refund sent' });
  });

  it('does nothing when the operator cancels the confirmation', async () => {
    confirmAction.mockResolvedValue(false);
    render(RefundRequiredView);
    await screen.findByText('Late payment');
    await fireEvent.click(screen.getByRole('button', { name: 'Resolve' }));
    expect(post).not.toHaveBeenCalled();
  });
});
