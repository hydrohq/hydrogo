<script lang="ts">
  import { type Ad, getMediaUrl, safeUrl } from '#lib';

  interface Props {
    ad?: Ad | null;
    zone?: string;
    class?: string;
  }

  let { ad = null, zone = '', class: className = '' }: Props = $props();
</script>

{#if ad}
  <div class="my-4 w-full flex justify-center {className}">
    {#if ad.custom_code}
      <!-- Sandboxed iframe to prevent stored XSS from arbitrary advertiser scripts -->
      <iframe
        srcdoc={`<!DOCTYPE html><html><head><meta charset="utf-8"><style>body{margin:0;padding:0;display:flex;justify-content:center;align-items:center;background:transparent;}</style></head><body>${ad.custom_code}</body></html>`}
        sandbox="allow-scripts allow-popups allow-forms"
        loading="lazy"
        title={ad.name || 'Sponsorlu Reklam'}
        class="border-0 overflow-hidden w-full max-w-[728px] min-h-[90px]"
      ></iframe>
    {:else if ad.banner_image}
      <a
        href={safeUrl(ad.target_url)}
        target="_blank"
        rel="noopener noreferrer sponsored"
        class="block group overflow-hidden rounded-lg transition-opacity hover:opacity-95"
      >
        <img
          src={getMediaUrl(ad.banner_image)}
          alt={ad.name}
          class="max-w-full h-auto mx-auto object-contain rounded"
          loading="lazy"
        />
      </a>
    {/if}
  </div>
{:else if zone}
  <!-- Placeholder slot for design preview if no active ad is placed in this zone -->
  <div class="my-4 w-full h-[90px] max-w-[728px] mx-auto bg-[var(--hp-surface-sunken)] border border-dashed border-[var(--hp-border-strong)] flex items-center justify-center text-xs text-[var(--hp-subtle)] uppercase tracking-wider {className}">
    Reklam Alanı · {zone}
  </div>
{/if}
