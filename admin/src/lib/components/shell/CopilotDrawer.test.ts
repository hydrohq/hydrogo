// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn()
}));

vi.mock('$lib/api/client', () => ({
  api: {
    get: getMock,
    post: postMock
  },
  ApiClientError: class extends Error {
    status: number;
    requestId?: string;
    constructor(message: string, status = 400, requestId?: string) {
      super(message);
      this.status = status;
      this.requestId = requestId;
    }
  }
}));

import CopilotDrawer from './CopilotDrawer.svelte';

describe('CopilotDrawer', () => {
  beforeEach(() => {
    getMock.mockReset();
    postMock.mockReset();
    getMock.mockResolvedValue({
      actions: [
        {
          id: 'act_101',
          tenant_id: 'ipek-butik',
          prompt: 'İpek Gömlek fiyatını 1450 TL yap',
          reply: 'Fiyat 1450 TL olarak güncellendi.',
          provider: 'gemini',
          model: 'gemini-2.5-flash',
          tokens_used: 48,
          operations: [
            {
              op: 'update',
              collection_id: 'boutique_products',
              record_id: 'rec_999'
            }
          ],
          undone: false,
          created_at: '2026-10-31T00:00:00Z'
        }
      ],
      governance: {
        tenant_id: 'ipek-butik',
        ai_enabled: true,
        ai_provider: 'gemini',
        ai_model: 'gemini-2.5-flash',
        has_custom_api_key: true,
        ai_monthly_token_limit: 500000,
        ai_tokens_used: 12500
      }
    });
  });

  it('loads action history and supports one-click Undo', async () => {
    render(CopilotDrawer, { open: true });

    await waitFor(() => {
      expect(screen.getByText('AI Mağaza Asistanı')).toBeTruthy();
      expect(screen.getByText('Fiyat 1450 TL olarak güncellendi.')).toBeTruthy();
    });

    postMock.mockResolvedValueOnce({
      action_log: {
        id: 'act_101',
        tenant_id: 'ipek-butik',
        prompt: 'İpek Gömlek fiyatını 1450 TL yap',
        reply: 'Fiyat 1450 TL olarak güncellendi.',
        provider: 'gemini',
        model: 'gemini-2.5-flash',
        tokens_used: 48,
        operations: [
          {
            op: 'update',
            collection_id: 'boutique_products',
            record_id: 'rec_999'
          }
        ],
        undone: true,
        created_at: '2026-10-31T00:00:00Z'
      }
    });

    const undoBtn = screen.getByText('↺ Geri Al (Undo)');
    await fireEvent.click(undoBtn);

    await waitFor(() => {
      expect(postMock).toHaveBeenCalledWith('/api/v1/admin/ai/actions/act_101/undo', {});
      expect(screen.getByText('↺ Geri alındı')).toBeTruthy();
    });
  });
});
