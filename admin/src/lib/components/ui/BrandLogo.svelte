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
      class="flex {iconSizes[size]} shrink-0 items-center justify-center rounded-lg bg-[#09090B] text-[#00ADD8] font-semibold shadow-xs border border-border/40"
    >
      <svg
        class="{svgSizes[size]}"
        viewBox="0 0 32 32"
        fill="none"
      >
        <path
          d="M10.4 17.4C12.2 16.2 14.1 16.2 16 17.4C17.9 18.6 19.8 18.6 21.6 17.4V24.2C21.6 25.4 20.6 26.4 19.4 26.4H12.6C11.4 26.4 10.4 25.4 10.4 24.2V17.4Z"
          fill="#00ADD8"
        />
        <path
          d="M10.4 17.4C12.2 16.2 14.1 16.2 16 17.4C17.9 18.6 19.8 18.6 21.6 17.4"
          stroke="#7DD3FC"
          stroke-width="1.2"
          stroke-linecap="round"
        />
        <path
          d="M13.5 6.2H18.5V9.8C18.5 10.7 19.1 11.4 19.9 11.9C21.3 12.7 22.2 14 22.2 15.6V24.2C22.2 25.8 20.9 27 19.3 27H12.7C11.1 27 9.8 25.8 9.8 24.2V15.6C9.8 14 10.7 12.7 12.1 11.9C12.9 11.4 13.5 10.7 13.5 9.8V6.2Z"
          stroke="#FFFFFF"
          stroke-width="1.8"
          stroke-linejoin="round"
        />
        <path
          d="M12.4 5.5H19.6"
          stroke="#FFFFFF"
          stroke-width="2"
          stroke-linecap="round"
        />
        <path
          d="M12.2 15.2V22.5"
          stroke="#FFFFFF"
          stroke-width="1.2"
          stroke-linecap="round"
          opacity="0.45"
        />
      </svg>
    </div>
  {/if}
  {#if showText}
    <span class="{textSizes[size]} font-semibold tracking-tight text-current truncate">
      {brandName}
    </span>
  {/if}
</div>
