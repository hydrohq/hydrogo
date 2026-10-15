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
    value = $bindable(''),
    id,
    class: className = '',
    ...restProps
  }: Props = $props();

  const inputId = $derived(id || `input-${Math.random().toString(36).substring(2, 9)}`);
</script>

<div class="w-full flex flex-col gap-1.5">
  {#if label}
    <label for={inputId} class="text-xs font-semibold uppercase tracking-wider text-slate-300">
      {label}
    </label>
  {/if}
  <input
    id={inputId}
    bind:value
    class="w-full rounded-lg bg-slate-900/90 border {error ? 'border-rose-500/80 focus:border-rose-500 focus:ring-rose-500/20' : 'border-slate-800 focus:border-sky-500 focus:ring-sky-500/20'} px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 transition-all duration-150 disabled:opacity-50 {className}"
    {...restProps}
  />
  {#if error}
    <p class="text-xs text-rose-400 font-medium">{error}</p>
  {:else if hint}
    <p class="text-xs text-slate-400">{hint}</p>
  {/if}
</div>
