// @vitest-environment jsdom
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock('$lib/api/client', () => ({
  api: { get, post },
  ApiClientError: class ApiClientError extends Error {
    status: number;
    requestId?: string;
    constructor(message: string, status = 500, requestId?: string) {
      super(message);
      this.status = status;
      this.requestId = requestId;
    }
  },
}));

const { confirmAction } = vi.hoisted(() => ({ confirmAction: vi.fn() }));
vi.mock('$lib/confirm', () => ({ confirmAction }));

import OrderInvoicePanel, { type InvoiceOrder } from './OrderInvoicePanel.svelte';

function makeOrder(overrides: Partial<InvoiceOrder> = {}): InvoiceOrder {
  return {
    id: 'order_1',
    payment_status: 'paid',
    invoice_status: 'none',
    ...overrides,
  };
}

describe('OrderInvoicePanel', () => {
  beforeEach(() => {
    get.mockReset();
    post.mockReset();
    confirmAction.mockReset().mockResolvedValue(true);
  });

  it('issues an invoice when operator confirms', async () => {
    get.mockResolvedValue({
      data: { configured: true, provider: 'invoicer', auto_issue_on_paid: false },
    });
    const issuedOrder: InvoiceOrder = {
      id: 'order_1',
      payment_status: 'paid',
      invoice_status: 'issued',
      invoice_provider: 'invoicer',
      invoice_no: 'order_1',
      invoice_guid: 'BH-GUID-1',
      invoice_url: 'https://invoice.example.test/BH-GUID-1',
      invoiced_at: '2026-11-13T18:00:00Z',
    };
    post.mockResolvedValueOnce({ data: issuedOrder });

    const onupdated = vi.fn();
    const onfeedback = vi.fn();
    render(OrderInvoicePanel, { props: { order: makeOrder(), canWrite: true, onupdated, onfeedback } });

    const issueBtn = await screen.findByRole('button', { name: 'Issue invoice / Send to the invoicing provider' });
    await fireEvent.click(issueBtn);

    await waitFor(() => {
      expect(confirmAction).toHaveBeenCalledTimes(1);
      expect(post).toHaveBeenCalledWith('/api/v1/admin/commerce/orders/order_1/invoice');
    });
    expect(onupdated).toHaveBeenCalledWith({ ...issuedOrder, invoice_error: '' });
    expect(onfeedback).toHaveBeenCalledWith(expect.objectContaining({ type: 'success' }));
  });

  it('shows GUID and open invoice link for issued orders', async () => {
    get.mockResolvedValue({
      data: { configured: true, provider: 'invoicer', auto_issue_on_paid: false },
    });

    render(OrderInvoicePanel, {
      props: {
        order: makeOrder({
          invoice_status: 'issued',
          invoice_provider: 'invoicer',
          invoice_guid: 'BH-GUID-99',
          invoice_url: 'https://invoice.example.test/BH-GUID-99',
        }),
        canWrite: true,
        onupdated: vi.fn(),
        onfeedback: vi.fn(),
      },
    });

    expect(await screen.findByText('BH-GUID-99')).toBeTruthy();
    const link = screen.getByRole('link', { name: 'Open / print invoice' });
    expect(link.getAttribute('href')).toBe('https://invoice.example.test/BH-GUID-99');
  });

  it('displays error message and retry button on failed invoice', async () => {
    get.mockResolvedValue({
      data: { configured: true, provider: 'invoicer', auto_issue_on_paid: false },
    });

    render(OrderInvoicePanel, {
      props: {
        order: makeOrder({
          invoice_status: 'failed',
          invoice_error: 'corporate invoice requires 10-digit vkn',
        }),
        canWrite: true,
        onupdated: vi.fn(),
        onfeedback: vi.fn(),
      },
    });

    expect(await screen.findByText('corporate invoice requires 10-digit vkn')).toBeTruthy();
    expect(await screen.findByRole('button', { name: 'Retry invoice / Send to the invoicing provider' })).toBeTruthy();
  });
});
