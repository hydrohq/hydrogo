<script lang="ts">
  import type { CodeSample } from '../docs/types';
  import CodeBlock from './CodeBlock.svelte';

  let { tabs }: { tabs: CodeSample[] } = $props();
  let active = $state(0);
  const uid = Math.random().toString(36).slice(2, 8);

  function onkey(e: KeyboardEvent) {
    let next = active;
    if (e.key === 'ArrowRight') next = (active + 1) % tabs.length;
    else if (e.key === 'ArrowLeft') next = (active - 1 + tabs.length) % tabs.length;
    else if (e.key === 'Home') next = 0;
    else if (e.key === 'End') next = tabs.length - 1;
    else return;
    e.preventDefault();
    active = next;
    document.getElementById(`${uid}-tab-${next}`)?.focus();
  }
</script>

<div class="my-5">
  <div role="tablist" aria-label="Code examples" class="flex gap-1 border-b border-line">
    {#each tabs as t, i}
      <button
        type="button"
        role="tab"
        id="{uid}-tab-{i}"
        aria-selected={active === i}
        aria-controls="{uid}-panel-{i}"
        tabindex={active === i ? 0 : -1}
        onclick={() => (active = i)}
        onkeydown={onkey}
        class="-mb-px border-b-2 px-3 py-2 text-sm font-medium {active === i ? 'border-accent text-accent' : 'border-transparent text-muted hover:text-fg'}"
      >
        {t.label}
      </button>
    {/each}
  </div>
  {#each tabs as t, i}
    {#if active === i}
      <div role="tabpanel" id="{uid}-panel-{i}" aria-labelledby="{uid}-tab-{i}" class="[&>div]:mt-0 [&>div]:rounded-t-none [&>div]:border-t-0">
        <CodeBlock code={t.code} lang={t.lang} title={t.title} />
      </div>
    {/if}
  {/each}
</div>
