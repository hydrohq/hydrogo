<script lang="ts">
  import { sections } from '../docs';
  import { docsHref } from '../router.svelte';

  let { section, page, onnavigate }: { section: string; page: string; onnavigate?: () => void } = $props();
</script>

<nav aria-label="Documentation" class="text-sm">
  {#each sections as s (s.id)}
    <div class="mb-6">
      <h2 class="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-faint">{s.title}</h2>
      <ul class="space-y-0.5">
        {#each s.pages as p (p.id)}
          {@const active = s.id === section && p.id === page}
          <li>
            <a
              href={docsHref(s.id, p.id)}
              onclick={onnavigate}
              aria-current={active ? 'page' : undefined}
              class="block rounded-md border-l-2 px-3 py-1.5 no-underline transition-colors {active
                ? 'border-accent bg-accent-soft font-medium text-accent'
                : 'border-transparent text-muted hover:bg-subtle hover:text-fg'}"
            >
              {p.title}
            </a>
          </li>
        {/each}
      </ul>
    </div>
  {/each}
</nav>
