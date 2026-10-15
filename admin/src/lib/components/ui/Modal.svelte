<script lang="ts">
  import type { Snippet } from 'svelte';
  import Button from './Button.svelte';

  interface Props {
    open: boolean;
    title: string;
    description?: string;
    onclose: () => void;
    children?: Snippet;
    footer?: Snippet;
  }

  let {
    open = $bindable(false),
    title,
    description,
    onclose,
    children,
    footer,
  }: Props = $props();

  function handleBackdrop(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onclose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) {
      onclose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <button
      type="button"
      class="fixed inset-0 bg-black/70 backdrop-blur-sm animate-in fade-in duration-150 cursor-default"
      onclick={onclose}
      aria-label="Close modal backdrop"
      tabindex="-1"
    ></button>
    <div
      class="relative w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh] z-10"
      role="dialog"
      aria-modal="true"
    >
      <div class="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
        <div>
          <h2 class="text-base font-semibold text-slate-100">{title}</h2>
          {#if description}
            <p class="text-xs text-slate-400 mt-0.5">{description}</p>
          {/if}
        </div>
        <Button variant="ghost" size="sm" onclick={onclose}>✕</Button>
      </div>

      <div class="p-6 overflow-y-auto">
        {#if children}
          {@render children()}
        {/if}
      </div>

      {#if footer}
        <div class="px-6 py-3.5 bg-slate-950/60 border-t border-slate-800/80 flex items-center justify-end gap-2.5">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
