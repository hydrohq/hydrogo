<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends HTMLButtonAttributes {
    variant?: 'primary' | 'secondary' | 'outline' | 'danger' | 'ghost';
    size?: 'sm' | 'md' | 'lg';
    loading?: boolean;
    children?: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    loading = false,
    class: className = '',
    disabled = false,
    children,
    ...restProps
  }: Props = $props();

  const variantClasses = {
    primary: 'bg-sky-600 hover:bg-sky-500 text-white shadow-sm shadow-sky-950/50 focus-visible:ring-sky-500',
    secondary: 'bg-slate-800 hover:bg-slate-700 text-slate-100 border border-slate-700/80 focus-visible:ring-slate-500',
    outline: 'bg-transparent hover:bg-slate-800/80 text-slate-200 border border-slate-700 focus-visible:ring-slate-500',
    danger: 'bg-rose-600 hover:bg-rose-500 text-white shadow-sm shadow-rose-950/50 focus-visible:ring-rose-500',
    ghost: 'bg-transparent hover:bg-slate-800/50 text-slate-300 hover:text-white',
  };

  const sizeClasses = {
    sm: 'text-xs px-2.5 py-1.5 rounded-md gap-1.5',
    md: 'text-sm px-3.5 py-2 rounded-lg gap-2',
    lg: 'text-base px-5 py-2.5 rounded-xl gap-2.5',
  };
</script>

<button
  class="inline-flex items-center justify-center font-medium transition-all duration-150 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus-visible:ring-2 {variantClasses[variant]} {sizeClasses[size]} {className}"
  disabled={disabled || loading}
  {...restProps}
>
  {#if loading}
    <svg class="animate-spin -ml-0.5 h-4 w-4 text-current" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
    </svg>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</button>
