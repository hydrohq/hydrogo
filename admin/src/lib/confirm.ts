import { writable } from 'svelte/store';

export interface ConfirmOptions {
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'destructive' | 'primary';
}

export interface ConfirmDialogState extends ConfirmOptions {
  open: boolean;
  loading?: boolean;
  resolve?: (value: boolean) => void;
}

const initialState: ConfirmDialogState = {
  open: false,
  title: '',
  description: '',
  confirmLabel: '',
  cancelLabel: '',
  variant: 'destructive',
  loading: false,
};

export const confirmDialogState = writable<ConfirmDialogState>(initialState);

/**
 * Programmatically opens a modal confirmation dialog anywhere in the HydroGo admin.
 * Returns a promise that resolves to true if the user confirmed, or false if cancelled.
 */
export function confirmAction(options: ConfirmOptions): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    confirmDialogState.set({
      ...options,
      open: true,
      loading: false,
      resolve,
    });
  });
}

/**
 * Resolves the currently active confirmation dialog.
 */
export function resolveConfirm(confirmed: boolean) {
  confirmDialogState.update((state) => {
    if (state.resolve) {
      state.resolve(confirmed);
    }
    return {
      ...initialState,
      open: false,
    };
  });
}

// Bind to window for global developer convenience and legacy script compatibility
if (typeof window !== 'undefined') {
  (window as any).confirmAction = confirmAction;
}
