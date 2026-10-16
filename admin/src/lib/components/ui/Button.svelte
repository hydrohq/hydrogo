<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends HTMLButtonAttributes {
    variant?: 'primary' | 'secondary' | 'outline' | 'danger' | 'ghost' | 'link';
    size?: 'sm' | 'md' | 'lg' | 'icon';
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

  const variantClasses: Record<string, string> = {
    primary: 'bg-primary text-primary-foreground shadow-xs hover:bg-primary/90 focus-visible:ring-ring',
    secondary: 'bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary/80 focus-visible:ring-ring',
    outline: 'border border-input bg-background shadow-xs hover:bg-accent hover:text-accent-foreground focus-visible:ring-ring',
    danger: 'bg-destructive text-destructive-foreground shadow-xs hover:bg-destructive/90 focus-visible:ring-destructive',
    ghost: 'hover:bg-accent hover:text-accent-foreground focus-visible:ring-ring',
    link: 'text-primary underline-offset-4 hover:underline focus-visible:ring-ring',
  };

  const sizeClasses: Record<string, string> = {
    sm: 'h-8 px-3 text-xs rounded-md gap-1.5',
    md: 'h-9 px-4 py-2 text-sm rounded-md gap-2',
    lg: 'h-10 px-6 py-2.5 text-base rounded-md gap-2.5',
    icon: 'h-9 w-9 rounded-md p-0 justify-center',
  };
</script>

<button
  class="inline-flex items-center justify-center font-medium transition-all duration-150 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-offset-background {variantClasses[variant]} {sizeClasses[size]} {className}"
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
