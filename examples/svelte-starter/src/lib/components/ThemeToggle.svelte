<script lang="ts">
  import { theme } from '../theme.svelte';

  const label = $derived(
    theme.choice === 'system' ? 'Theme: follows system. Click to use light.' : theme.choice === 'light' ? 'Theme: light. Click to use dark.' : 'Theme: dark. Click to follow system.',
  );
</script>

<button
  type="button"
  onclick={() => theme.cycle()}
  class="inline-flex size-9 items-center justify-center rounded-md text-muted hover:bg-subtle hover:text-fg"
  aria-label={label}
  title={label}
>
  {#if theme.choice === 'system'}
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="2" y="3" width="20" height="14" rx="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" /></svg>
  {:else if theme.choice === 'light'}
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></svg>
  {:else}
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" /></svg>
  {/if}
</button>
