<script lang="ts">
  import { enhance } from '$app/forms';
  import { messages, type Lang } from '$lib/i18n';
  import { isOverdue } from '$lib/pipeline';
  type Activity = { id: string; subject?: unknown; kind?: unknown; due?: unknown; done?: unknown };
  let { items, lang }: { items: Activity[]; lang: Lang } = $props();
  const t = $derived(messages[lang]);
</script>

{#if items.length === 0}
  <p class="muted">{t.empty}</p>
{:else}
  <ul class="timeline">
    {#each items as item}
      <li>
        <span class:done={item.done}>{item.subject} <span class="muted">· {t.kinds[item.kind as keyof typeof t.kinds] ?? item.kind}</span></span>
        <span>
          {#if item.due}<span class:overdue={!item.done && isOverdue(item.due)}>{new Date(item.due as string).toLocaleString(lang)}</span>{/if}
          {#if !item.done}
            <form method="POST" action="?/done" use:enhance style="display:inline"><input type="hidden" name="id" value={item.id} /><button>{t.markDone}</button></form>
          {/if}
        </span>
      </li>
    {/each}
  </ul>
{/if}
