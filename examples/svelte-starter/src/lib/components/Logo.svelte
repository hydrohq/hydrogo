<script lang="ts">
  let { size = 'md' }: { size?: 'md' | 'lg' } = $props();
</script>

<span class="inline-flex items-center gap-2.5 font-semibold tracking-tight text-fg {size === 'lg' ? 'text-xl' : 'text-base'}">
  <span class="inline-flex items-center justify-center rounded-lg border border-accent/30 bg-accent-soft text-accent {size === 'lg' ? 'size-10' : 'size-7'}">
    <svg class={size === 'lg' ? 'size-5' : 'size-4'} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <polygon points="12 2 2 7 12 12 22 7 12 2" />
      <polyline points="2 17 12 22 22 17" />
      <polyline points="2 12 12 17 22 12" />
    </svg>
  </span>
  <span>Hydro<span class="text-accent">Go</span></span>
</span>
