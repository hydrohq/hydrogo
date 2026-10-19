<script lang="ts">
  import { docsHref } from '../router.svelte';

  let {
    section,
    page,
    headings,
    activeId,
  }: { section: string; page: string; headings: { id: string; text: string; level: 2 | 3 }[]; activeId: string } = $props();
</script>

{#if headings.length > 0}
  <nav aria-label="On this page" class="text-sm">
    <h2 class="mb-3 text-xs font-semibold uppercase tracking-wider text-faint">On this page</h2>
    <ul class="space-y-1 border-l border-line">
      {#each headings as h (h.id)}
        <li>
          <a
            href={docsHref(section, page, h.id)}
            aria-current={activeId === h.id ? 'location' : undefined}
            class="-ml-px block border-l-2 py-1 no-underline transition-colors {h.level === 3 ? 'pl-6' : 'pl-3'} {activeId === h.id
              ? 'border-accent font-medium text-accent'
              : 'border-transparent text-muted hover:border-line-strong hover:text-fg'}"
          >
            {h.text}
          </a>
        </li>
      {/each}
    </ul>
  </nav>
{/if}
