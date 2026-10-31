import { describe, it, expect } from 'vitest';
import { get } from 'svelte/store';
import { confirmAction, resolveConfirm, confirmDialogState } from './confirm';

describe('confirmAction helper and store', () => {
  it('opens dialog with options and resolves true when confirmed', async () => {
    const promise = confirmAction({
      title: 'Mağazayı Askıya Al',
      description: 'Ziyaretçiler erişemeyecektir.',
      confirmLabel: 'Askıya Al',
      variant: 'destructive',
    });

    const state = get(confirmDialogState);
    expect(state.open).toBe(true);
    expect(state.title).toBe('Mağazayı Askıya Al');
    expect(state.confirmLabel).toBe('Askıya Al');
    expect(state.variant).toBe('destructive');

    // User confirms
    resolveConfirm(true);

    const result = await promise;
    expect(result).toBe(true);
    expect(get(confirmDialogState).open).toBe(false);
  });

  it('resolves false when cancelled', async () => {
    const promise = confirmAction({
      title: 'İptal Testi',
      confirmLabel: 'Onayla',
    });

    expect(get(confirmDialogState).open).toBe(true);

    // User cancels
    resolveConfirm(false);

    const result = await promise;
    expect(result).toBe(false);
    expect(get(confirmDialogState).open).toBe(false);
  });
});
