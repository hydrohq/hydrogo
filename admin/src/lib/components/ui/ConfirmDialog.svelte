<script lang="ts">
  interface Props {
    open: boolean;
    title: string;
    description?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: 'destructive' | 'primary';
    loading?: boolean;
    onconfirm: () => void | Promise<void>;
    oncancel: () => void;
  }

  let {
    open = false,
    title,
    description = 'Are you sure you want to proceed with this action? This cannot be undone.',
    confirmLabel = 'Delete',
    cancelLabel = 'Cancel',
    variant = 'destructive',
    loading = false,
    onconfirm,
    oncancel,
  }: Props = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open && !loading) {
      oncancel();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-0">
    <!-- Backdrop with smooth blur -->
    <div
      class="fixed inset-0 bg-black/60 backdrop-blur-xs transition-opacity animate-in fade-in"
      onclick={() => { if (!loading) oncancel(); }}
      aria-hidden="true"
    ></div>

    <!-- Dialog Card -->
    <div
      class="relative z-10 w-full max-w-md transform overflow-hidden rounded-xl border border-border bg-card p-6 text-card-foreground shadow-xl transition-all sm:my-8 animate-in fade-in zoom-in-95 duration-150"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
    >
      <div class="flex items-start gap-4">
        <!-- Icon Badge -->
        {#if variant === 'destructive'}
          <div class="size-10 rounded-full bg-destructive/10 text-destructive flex items-center justify-center shrink-0">
            <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 6h18"></path>
              <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path>
              <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path>
              <line x1="10" y1="11" x2="10" y2="17"></line>
              <line x1="14" y1="11" x2="14" y2="17"></line>
            </svg>
          </div>
        {:else}
          <div class="size-10 rounded-full bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
          </div>
        {/if}

        <div class="flex-1 space-y-1.5">
          <h3 id="dialog-title" class="text-base font-semibold text-foreground leading-none">
            {title}
          </h3>
          <p class="text-xs text-muted-foreground leading-relaxed">
            {description}
          </p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="mt-6 flex items-center justify-end gap-2.5">
        <button
          type="button"
          disabled={loading}
          onclick={oncancel}
          class="h-9 px-4 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs disabled:opacity-50"
        >
          {cancelLabel}
        </button>

        <button
          type="button"
          disabled={loading}
          onclick={onconfirm}
          class="h-9 px-4 rounded-md text-xs font-medium transition-colors cursor-pointer shadow-xs flex items-center gap-1.5 disabled:opacity-50 {variant === 'destructive' ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90' : 'bg-primary text-primary-foreground hover:bg-primary/90'}"
        >
          {#if loading}
            <div class="size-3 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
          {/if}
          <span>{confirmLabel}</span>
        </button>
      </div>
    </div>
  </div>
{/if}
