<script lang="ts">
  import type { CalloutKind } from '../docs/types';
  import Inline from './Inline.svelte';

  let { kind, title, text }: { kind: CalloutKind; title?: string; text: string } = $props();
  const labels: Record<CalloutKind, string> = { note: 'Note', tip: 'Tip', warning: 'Warning' };
</script>

<aside
  class="my-5 rounded-lg border-l-4 px-4 py-3 text-[15px] leading-relaxed"
  style="border-color: var(--{kind === 'warning' ? 'warn' : kind}); background: var(--{kind === 'warning' ? 'warn' : kind}-bg)"
  aria-label={labels[kind]}
>
  <p class="mb-1 flex items-center gap-2 text-sm font-semibold" style="color: var(--{kind === 'warning' ? 'warn' : kind})">
    {#if kind === 'warning'}
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" /></svg>
    {:else if kind === 'tip'}
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 18h6M10 22h4M12 2a7 7 0 0 0-4 12.7c.6.5 1 1.3 1 2.3h6c0-1 .4-1.8 1-2.3A7 7 0 0 0 12 2z" /></svg>
    {:else}
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></svg>
    {/if}
    {title ?? labels[kind]}
  </p>
  <p class="text-fg"><Inline {text} /></p>
</aside>
