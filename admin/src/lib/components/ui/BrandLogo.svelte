<script lang="ts">
  import { brandingStore } from '$lib/stores/branding.svelte';
  interface Props {
    size?: 'sm' | 'md' | 'lg' | 'xl';
    showIcon?: boolean;
    showText?: boolean;
    class?: string;
  }

  let { size = 'md', showIcon = true, showText = true, class: className = '' }: Props = $props();

  const brand = $derived(brandingStore.branding);
  const brandName = $derived(brand?.name || 'HydroGo');

  const iconSizes = {
    sm: 'size-6',
    md: 'size-7',
    lg: 'size-10',
    xl: 'size-14 sm:size-16',
  };

  const svgSizes = {
    sm: 'size-3.5',
    md: 'size-4',
    lg: 'size-5',
    xl: 'size-7 sm:size-8',
  };

  const textSizes = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-xl',
    xl: 'text-4xl sm:text-5xl',
  };
</script>

<div class="flex items-center gap-2.5 select-none {className}">
  {#if showIcon && brand?.logo_url}
    <img src={brand.logo_url} alt="" class="{iconSizes[size]} shrink-0 rounded-lg object-contain" />
  {:else if showIcon}
    <div
      class="flex {iconSizes[size]} shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground font-semibold shadow-xs"
    >
      <svg
        class="{svgSizes[size]}"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
        <polyline points="2 17 12 22 22 17"></polyline>
        <polyline points="2 12 12 17 22 12"></polyline>
      </svg>
    </div>
  {/if}
  {#if showText}
    <span class="{textSizes[size]} font-semibold tracking-tight text-current truncate">
      {brandName}
    </span>
  {/if}
</div>
