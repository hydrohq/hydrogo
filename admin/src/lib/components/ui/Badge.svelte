<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'danger' | 'destructive' | 'info' | 'outline';
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
    default: 'bg-primary text-primary-foreground border-transparent shadow-xs',
    primary: 'bg-primary text-primary-foreground border-transparent shadow-xs',
    secondary: 'bg-secondary text-secondary-foreground border-transparent',
    success: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-500/20',
    warning: 'bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-500/20',
    danger: 'bg-destructive/10 text-destructive border-destructive/20',
    destructive: 'bg-destructive/10 text-destructive border-destructive/20',
    info: 'bg-muted text-muted-foreground border-border',
    outline: 'border-border text-foreground bg-transparent',
  };

  const sizeStyles: Record<string, string> = {
    sm: 'text-[10px] px-2 py-0.5',
    md: 'text-xs px-2.5 py-0.5',
  };
</script>

<span class="inline-flex items-center gap-1.5 font-medium rounded-md border transition-colors {variantStyles[variant] || variantStyles.default} {sizeStyles[size]} {className}">
  {#if children}
    {@render children()}
  {/if}
</span>
