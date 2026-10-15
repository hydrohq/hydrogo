<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'danger' | 'info' | 'outline';
    size?: 'sm' | 'md';
    class?: string;
    children?: Snippet;
  }

  let {
    variant = 'default',
    size = 'md',
    class: className = '',
    children
  }: Props = $props();

  const variantStyles: Record<string, string> = {
    default: 'bg-secondary text-secondary-foreground border-border/80',
    primary: 'bg-primary/15 text-blue-400 border-primary/30',
    secondary: 'bg-secondary text-secondary-foreground border-transparent',
    success: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
    warning: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
    danger: 'bg-destructive/15 text-rose-400 border-destructive/30',
    info: 'bg-sky-500/15 text-sky-400 border-sky-500/30',
    outline: 'border-border text-foreground bg-transparent',
  };

  const sizeStyles: Record<string, string> = {
    sm: 'text-[10px] px-2 py-0.5',
    md: 'text-xs px-2.5 py-0.5',
  };
</script>

<span class="inline-flex items-center gap-1.5 font-medium rounded-full border transition-colors {variantStyles[variant]} {sizeStyles[size]} {className}">
  {#if children}
    {@render children()}
  {/if}
</span>
