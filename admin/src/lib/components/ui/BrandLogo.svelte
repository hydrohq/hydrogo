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
  const hasCustomName = $derived(Boolean(brand?.name?.trim()));
  const brandName = $derived(brand?.name?.trim() || 'HydroGo');
  const brandInitial = $derived(brandName.charAt(0).toUpperCase());

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
  {:else if showIcon && !hasCustomName}
    <div
      class="flex {iconSizes[size]} shrink-0 items-center justify-center rounded-lg bg-[#09090B] text-[#00ADD8] font-semibold shadow-xs border border-border/40"
    >
      <svg
        class="{svgSizes[size]}"
        viewBox="0 0 24 24"
        fill="none"
      >
        <path
          d="M12 2.5C12 2.5 5.5 10.2 5.5 15C5.5 18.59 8.41 21.5 12 21.5C15.59 21.5 18.5 18.59 18.5 15C18.5 10.2 12 2.5 12 2.5Z"
          fill="#00ADD8"
        />
        <path
          d="M9.4 15.8C9.4 14 10.4 11.7 11.7 9.8"
          stroke="#FFFFFF"
          stroke-width="1.6"
          stroke-linecap="round"
          opacity="0.85"
        />
      </svg>
    </div>
  {:else if showIcon && !showText && hasCustomName}
    <div
      class="flex {iconSizes[size]} shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground font-semibold shadow-xs border border-border/40"
      aria-hidden="true"
    >
      <span class="{textSizes[size]} leading-none">{brandInitial}</span>
    </div>
  {/if}
  {#if showText}
    <span class="{textSizes[size]} font-semibold tracking-tight text-current truncate">
      {brandName}
    </span>
  {/if}
</div>
