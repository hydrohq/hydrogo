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

import OrderShippingPanel, { type ShippingOrder } from './OrderShippingPanel.svelte';

function makeOrder(overrides: Partial<ShippingOrder> = {}): ShippingOrder {
  return {
    id: 'order_1',
    payment_status: 'paid',
    fulfillment_status: 'unfulfilled',
    ...overrides,
  };
}

describe('OrderShippingPanel', () => {
  beforeEach(() => {
    get.mockReset();
    post.mockReset();
    confirmAction.mockReset().mockResolvedValue(true);
  });

  it('fetches Geliver offers, selects an offer, and creates shipment barcode', async () => {
    get.mockResolvedValue({
      data: {
        configured: true,
        default_carrier: 'geliver',
        carriers: [{ key: 'geliver', supports_offers: true, supports_cancel: true }],
      },
    });
    post
      .mockResolvedValueOnce({
        data: {
          shipment_id: 'glv_ship_1',
          cheapest_id: 'offer_1',
          fastest_id: 'offer_2',
          offers: [
            { id: 'offer_1', carrier: 'yurtici', carrier_name: 'Yurtici Kargo', amount: '75.00', currency: 'TRY' },
            { id: 'offer_2', carrier: 'aras', carrier_name: 'Aras Kargo', service_code: 'EXPRESS', amount: '90.00', currency: 'TRY', estimated_delivery: '1 gun' },
          ],
        },
      })
      .mockResolvedValueOnce({
        data: {
          ...makeOrder(),
          fulfillment_status: 'shipped',
          shipping_carrier: 'aras',
          shipping_tracking_number: 'ARS12345',
          shipping_shipment_id: 'glv_ship_1',
          shipping_label_url: 'https://labels.example.test/glv_ship_1.pdf',
        },
      });

    const onupdated = vi.fn();
    const onfeedback = vi.fn();
    render(OrderShippingPanel, { props: { order: makeOrder(), canWrite: true, onupdated, onfeedback } });

    const fetchBtn = await screen.findByRole('button', { name: 'Get carrier offers' });
    await fireEvent.click(fetchBtn);

    expect(post).toHaveBeenNthCalledWith(1, '/api/v1/admin/commerce/orders/order_1/shipment/offers', { carrier: 'geliver' });
    expect(await screen.findByText('Aras Kargo')).toBeTruthy();

    const radios = screen.getAllByRole('radio') as HTMLInputElement[];
    expect(radios).toHaveLength(2);
    expect(radios[0].checked).toBe(true);
    await fireEvent.click(radios[1]);

    const acceptBtn = screen.getByRole('button', { name: 'Create barcode with selected offer' });
    await fireEvent.click(acceptBtn);

    await waitFor(() => {
      expect(post).toHaveBeenNthCalledWith(2, '/api/v1/admin/commerce/orders/order_1/shipment', {
        carrier: 'geliver',
        offer_id: 'offer_2',
        shipment_id: 'glv_ship_1',
      });
    });
    expect(onupdated).toHaveBeenCalledWith(expect.objectContaining({
      id: 'order_1',
      fulfillment_status: 'shipped',
      shipping_tracking_number: 'ARS12345',
    }));
  });

  it('shows barcode PDF link on shipped orders and cancels shipment with confirmation', async () => {
    get.mockResolvedValue({
      data: {
        configured: true,
        default_carrier: 'geliver',
        carriers: [{ key: 'geliver', supports_offers: true, supports_cancel: true }],
      },
    });
    const omittedCancelOrder = {
      id: 'order_1',
      payment_status: 'paid' as const,
      fulfillment_status: 'unfulfilled' as const,
      shipped_at: '0001-01-01T00:00:00Z',
    };
    post.mockResolvedValueOnce({ data: omittedCancelOrder });

    const shippedOrder = makeOrder({
      fulfillment_status: 'shipped',
      shipping_carrier: 'yurtici',
      shipping_tracking_number: 'YK999',
      shipping_tracking_url: 'https://track.example.test/YK999',
      shipping_shipment_id: 'glv_ship_9',
      shipping_label_url: 'https://labels.example.test/glv_ship_9.pdf',
      shipped_at: '2026-11-13T10:00:00Z',
    });

    const onupdated = vi.fn();
    const onfeedback = vi.fn();
    const view = render(OrderShippingPanel, { props: { order: shippedOrder, canWrite: true, onupdated, onfeedback } });

    const labelLink = await screen.findByRole('link', { name: 'Open / print barcode PDF' });
    expect(labelLink.getAttribute('href')).toBe('https://labels.example.test/glv_ship_9.pdf');

    const cancelBtn = screen.getByRole('button', { name: 'Cancel shipment' });
    await fireEvent.click(cancelBtn);

    await waitFor(() => {
      expect(confirmAction).toHaveBeenCalledTimes(1);
      expect(post).toHaveBeenCalledWith('/api/v1/admin/commerce/orders/order_1/shipment/cancel');
    });
    const updatedArg = onupdated.mock.calls[0]?.[0] as ShippingOrder;
    expect(updatedArg).toEqual(expect.objectContaining({
      id: 'order_1',
      fulfillment_status: 'unfulfilled',
      shipping_carrier: '',
      shipping_tracking_number: '',
      shipping_tracking_url: '',
      shipping_shipment_id: '',
      shipping_label_url: '',
    }));

    await view.rerender({ order: updatedArg, canWrite: true, onupdated, onfeedback });
    expect(screen.queryByRole('link', { name: 'Open / print barcode PDF' })).toBeNull();
    expect(await screen.findByRole('button', { name: 'Get carrier offers' })).toBeTruthy();
  });

  it('falls back to manual carrier form when Geliver is not configured', async () => {
    get.mockResolvedValue({
      data: {
        configured: true,
        default_carrier: 'manual',
        carriers: [{ key: 'manual', supports_offers: false, supports_cancel: false }],
      },
    });
    const shipped = makeOrder({
      fulfillment_status: 'shipped',
      shipping_carrier: 'Aras Kargo',
      shipping_tracking_number: 'TRK-42',
    });
    post.mockResolvedValueOnce({ data: shipped });

    const onupdated = vi.fn();
    const onfeedback = vi.fn();
    render(OrderShippingPanel, { props: { order: makeOrder(), canWrite: true, onupdated, onfeedback } });

    const carrierInput = await screen.findByLabelText('Carrier (optional, e.g. Aras Kargo)');
    const trackingInput = screen.getByLabelText('Tracking number (for carriers booked outside)');
    await fireEvent.input(carrierInput, { target: { value: 'Aras Kargo' } });
    await fireEvent.input(trackingInput, { target: { value: 'TRK-42' } });

    const shipBtn = screen.getByRole('button', { name: 'Hand to carrier' });
    await fireEvent.click(shipBtn);

    await waitFor(() => {
      expect(confirmAction).toHaveBeenCalledTimes(1);
      expect(post).toHaveBeenCalledWith('/api/v1/admin/commerce/orders/order_1/shipment', {
        carrier: 'Aras Kargo',
        tracking_number: 'TRK-42',
      });
    });
    expect(onupdated).toHaveBeenCalledWith(shipped);
  });
});
