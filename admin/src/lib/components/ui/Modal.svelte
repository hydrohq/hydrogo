<script lang="ts">
  import type { Snippet } from 'svelte';
  import Button from './Button.svelte';

  interface Props {
    open: boolean;
    title: string;
    description?: string;
    onclose?: () => void;
    onClose?: () => void;
    children?: Snippet;
    footer?: Snippet;
  }

  let {
    open = $bindable(false),
    title,
    description,
    onclose,
    onClose,
    children,
    footer,
  }: Props = $props();

  function close() {
    if (onclose) onclose();
    else if (onClose) onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) {
      close();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <button
      type="button"
      class="fixed inset-0 bg-black/80 backdrop-blur-sm transition-opacity duration-150 cursor-default"
      onclick={close}
      aria-label="Close modal backdrop"
      tabindex="-1"
    ></button>
    <div
      class="relative w-full max-w-lg bg-card border border-border text-card-foreground rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh] z-10 animate-in fade-in zoom-in-95 duration-150"
      role="dialog"
      aria-modal="true"
    >
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <div>
          <h2 class="text-base font-semibold tracking-tight text-foreground">{title}</h2>
          {#if description}
            <p class="text-xs text-muted-foreground mt-0.5">{description}</p>
          {/if}
        </div>
        <Button variant="ghost" size="icon" class="h-8 w-8 text-muted-foreground hover:text-foreground" onclick={close} aria-label="Close modal">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </Button>
      </div>

      <div class="p-6 overflow-y-auto">
        {#if children}
          {@render children()}
        {/if}
      </div>

      {#if footer}
        <div class="px-6 py-3.5 bg-muted/40 border-t border-border flex items-center justify-end gap-2.5">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
