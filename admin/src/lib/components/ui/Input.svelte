<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props extends HTMLInputAttributes {
    label?: string;
    error?: string;
    hint?: string;
    value?: any;
  }

  let {
    label,
    error,
    hint,
    value = $bindable(),
    id,
    class: className = '',
    ...restProps
  }: Props = $props();

  const inputId = $derived(id || `input-${Math.random().toString(36).substring(2, 9)}`);
</script>

<div class="w-full flex flex-col gap-1.5">
  {#if label}
    <label for={inputId} class="text-xs font-medium text-foreground/90">
      {label}
    </label>
  {/if}
  <input
    id={inputId}
    bind:value
    class="flex h-9 w-full rounded-md border {error ? 'border-destructive focus-visible:ring-destructive' : 'border-input focus-visible:ring-ring'} bg-background px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 disabled:cursor-not-allowed disabled:opacity-50 {className}"
    {...restProps}
  />
  {#if error}
    <p class="text-xs text-destructive font-medium">{error}</p>
  {:else if hint}
    <p class="text-xs text-muted-foreground">{hint}</p>
  {/if}
</div>
