<script lang="ts">
  import { parseInline } from '../docs/inline';

  let { text }: { text: string } = $props();
  const segments = $derived(parseInline(text));
</script>

{#each segments as s}
  {#if s.kind === 'code'}
    <code class="rounded border border-line bg-subtle px-1.5 py-0.5 font-mono text-[0.85em] text-fg break-words">{s.text}</code>
  {:else if s.kind === 'bold'}
    <strong class="font-semibold text-fg">{s.text}</strong>
  {:else if s.kind === 'link'}
    <a
      href={s.href}
      class="font-medium text-accent underline decoration-accent/30 underline-offset-2 hover:decoration-accent"
      {...s.external ? { target: '_blank', rel: 'noopener noreferrer' } : {}}>{s.text}</a
    >
  {:else}{s.text}{/if}
{/each}
