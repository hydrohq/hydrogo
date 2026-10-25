const FOCUSABLE = 'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])';
const stack: HTMLElement[] = [];

/**
 * Svelte action for dialogs: focus moves inside on open, Tab stays inside,
 * Escape closes (only the top-most dialog) and focus returns to the opener.
 */
export function modal(node: HTMLElement, onclose: () => void) {
  stack.push(node);
  const opener = document.activeElement as HTMLElement | null;
  const items = () => [...node.querySelectorAll<HTMLElement>(FOCUSABLE)].filter((e) => e.offsetParent !== null);
  queueMicrotask(() => (items()[0] ?? node).focus());

  const key = (e: KeyboardEvent) => {
    if (stack[stack.length - 1] !== node) return;
    if (e.key === 'Escape') {
      e.stopPropagation();
      onclose();
    } else if (e.key === 'Tab') {
      const f = items();
      if (!f.length) return;
      const first = f[0];
      const last = f[f.length - 1];
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    }
  };
  document.addEventListener('keydown', key);
  return {
    update(fn: () => void) { onclose = fn; },
    destroy() {
      document.removeEventListener('keydown', key);
      stack.splice(stack.indexOf(node), 1);
      opener?.focus?.();
    }
  };
}
